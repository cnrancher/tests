package huawei

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/cce-operator/pkg/utils"
	ecs_model "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	rancherHuawei "github.com/rancher/shepherd/clients/huawei"
	"github.com/rancher/shepherd/clients/rancher"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/nodes"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	nodeBaseName = "rancher-automation"
)

var backoff = wait.Backoff{
	Duration: 30 * time.Second,
	Steps:    20, // 10min
}

// CreateNodes creates `quantityPerPool[n]` number of huawei ecs instances
func CreateNodes(client *rancher.Client, rolesPerPool []string, quantityPerPool []int32) ([]*nodes.Node, error) {
	hwClient, err := client.GetHuaweiClient()
	if err != nil {
		return nil, err
	}
	runningReservations := []*ecs_model.CreateServersResponse{}
	reservationConfigs := []*rancherHuawei.ECSConfig{}
	for i := len(quantityPerPool) - 1; i >= 0; i-- {
		config := MatchRoleToConfig(rolesPerPool[i], hwClient.ClientConfig.HuaweiConfig)
		if config == nil {
			return nil, errors.New("No matching nodesAndRole for HuaweiConfig with role:" + rolesPerPool[i])
		}
		request := &ecs_model.CreateServersRequest{
			Body: &ecs_model.CreateServersRequestBody{
				DryRun: utils.Pointer(false),
				Server: &ecs_model.PrePaidServer{
					ImageRef:  config.ImageRef,
					FlavorRef: config.FlavorRef,
					Name:      namegen.AppendRandomString(nodeBaseName),
					KeyName:   &config.KeyName,
					Vpcid:     config.VpcID,
					Nics: []ecs_model.PrePaidServerNic{
						{
							SubnetId: config.SubnetID,
						},
					},
					Publicip: &ecs_model.PrePaidServerPublicip{
						Eip: &ecs_model.PrePaidServerEip{
							Iptype: config.EIPType,
							Bandwidth: &ecs_model.PrePaidServerEipBandwidth{
								Size:       &config.BandwidthSize,
								Chargemode: &config.BandwidthChargeMode,
								// Sharetype,
							},
						},
					},
					RootVolume: &ecs_model.PrePaidServerRootVolume{
						// Volumetype,
						Size: &config.RootVolumeSize,
					},
					DataVolumes:    nil,
					SecurityGroups: nil,
					UserData:       &config.UserData,
				},
			},
		}
		switch config.BandwidthShareType {
		case "WHOLE":
			request.Body.Server.Publicip.Eip.Bandwidth.Sharetype = ecs_model.GetPrePaidServerEipBandwidthSharetypeEnum().WHOLE
		default:
			request.Body.Server.Publicip.Eip.Bandwidth.Sharetype = ecs_model.GetPrePaidServerEipBandwidthSharetypeEnum().PER
		}
		switch config.RootVolumeType {
		case "SATA":
			request.Body.Server.RootVolume.Volumetype = ecs_model.GetPrePaidServerRootVolumeVolumetypeEnum().SATA
		case "SAS":
			request.Body.Server.RootVolume.Volumetype = ecs_model.GetPrePaidServerRootVolumeVolumetypeEnum().SAS
		case "GPSSD":
			request.Body.Server.RootVolume.Volumetype = ecs_model.GetPrePaidServerRootVolumeVolumetypeEnum().GPSSD
		case "ESSD":
			request.Body.Server.RootVolume.Volumetype = ecs_model.GetPrePaidServerRootVolumeVolumetypeEnum().ESSD
		default:
			request.Body.Server.RootVolume.Volumetype = ecs_model.GetPrePaidServerRootVolumeVolumetypeEnum().SSD
		}
		dataVolume := []ecs_model.PrePaidServerDataVolume{
			{
				Size: config.DataVolumeSize,
			},
		}
		switch config.DataVolumeType {
		case "SATA":
			dataVolume[0].Volumetype = ecs_model.GetPrePaidServerDataVolumeVolumetypeEnum().SATA
		case "SAS":
			dataVolume[0].Volumetype = ecs_model.GetPrePaidServerDataVolumeVolumetypeEnum().SAS
		case "GPSSD":
			dataVolume[0].Volumetype = ecs_model.GetPrePaidServerDataVolumeVolumetypeEnum().GPSSD
		case "ESSD":
			dataVolume[0].Volumetype = ecs_model.GetPrePaidServerDataVolumeVolumetypeEnum().ESSD
		default:
			dataVolume[0].Volumetype = ecs_model.GetPrePaidServerDataVolumeVolumetypeEnum().SSD
		}
		request.Body.Server.DataVolumes = &dataVolume
		if config.SecurityGroups != nil {
			sgs := []ecs_model.PrePaidServerSecurityGroup{}
			for _, sg := range config.SecurityGroups {
				sgs = append(sgs, ecs_model.PrePaidServerSecurityGroup{
					Id: utils.Pointer(sg),
				})
			}
			request.Body.Server.SecurityGroups = &sgs
		}
		reservation, err := hwClient.Client.CreateServers(request)
		if err != nil {
			return nil, err
		}
		if reservation == nil || reservation.ServerIds == nil {
			logrus.Warnf("skip reservation: %+v, CreateServers returns invalid value", reservation)
		}
		logrus.Infof("create huawei ecs instance [%v] %v",
			request.Body.Server.Name, *reservation.ServerIds)
		runningReservations = append(runningReservations, reservation)
		reservationConfigs = append(reservationConfigs, config)
	}
	// wait for create instance jobs started
	time.Sleep(time.Second * 10)
	hwNodes := []*nodes.Node{}
	for i := 0; i < len(quantityPerPool); i++ {
		serverIDs := runningReservations[i].ServerIds
		for _, id := range *serverIDs {
			var (
				publicIP  string
				privateIP string
				server    *ecs_model.ShowServerResponse
			)
			// wait for server initialized
			if err = wait.ExponentialBackoff(backoff, func() (bool, error) {
				var err error
				server, err = hwClient.Client.ShowServer(&ecs_model.ShowServerRequest{
					ServerId: id,
				})
				if err != nil {
					return false, err
				}
				if server == nil || server.Server == nil {
					logrus.Warnf("ShowServer returns invalid data, retry")
					return false, nil
				}
				if server.Server.Status != "ACTIVE" {
					logrus.Infof("wait for instance [%s] status [%s]",
						server.Server.Name, server.Server.Status)
					return false, nil
				}
				for _, addrs := range server.Server.Addresses {
					for _, addr := range addrs {
						if addr.OSEXTIPStype == nil {
							logrus.Warnf("Skip %v: OSEXTIPStype is nil", addr.Addr)
							continue
						}
						switch addr.OSEXTIPStype.Value() {
						case "floating":
							publicIP = addr.Addr
						case "fixed":
							privateIP = addr.Addr
						}
					}
				}
				if publicIP == "" || privateIP == "" {
					logrus.Infof("wait for instance [%s] create public IP",
						server.Server.Name)
					return false, nil
				}
				logrus.Infof("instance [%s] [%s] is running",
					server.Server.Name, id)
				return true, nil
			}); err != nil {
				return nil, err
			}
			logrus.Infof("public IP of node [%v]: %v", server.Server.Name, publicIP)
			sshKey, err := nodes.GetSSHKey(reservationConfigs[i].KeyName)
			if err != nil {
				return nil, err
			}
			node := &nodes.Node{
				NodeID:           id,
				PublicIPAddress:  publicIP,
				PrivateIPAddress: privateIP,
				SSHUser:          reservationConfigs[i].SSHUser,
				SSHKey:           sshKey,
			}
			// wait for docker installed by cloud-init
			if err = wait.ExponentialBackoff(backoff, func() (bool, error) {
				if os.Getenv("SKIP_HUAWEI_NODE_WAIT_DOCKER") == "true" {
					logrus.Infof("SKIP_HUAWEI_NODE_WAIT_DOCKER is set, skip waiting for docker")
					return true, nil
				}
				_, err := node.ExecuteCommand("docker ps")
				if err != nil {
					logrus.Infof("wait for server [%v] install docker", server.Server.Name)
					return false, nil
				}
				return true, nil
			}); err != nil {
				return nil, err
			}
			// re-reverse the list so that the order is corrected
			hwNodes = append([]*nodes.Node{node}, hwNodes...)
		}
	}
	client.Session.RegisterCleanupFunc(func() error {
		return DeleteNodes(client, hwNodes)
	})
	return hwNodes, nil
}

// MatchRoleToConfig matches the role of nodesAndRoles to the ECSConfig that allows this role.
func MatchRoleToConfig(poolRole string, huaweiConfigs []rancherHuawei.ECSConfig) *rancherHuawei.ECSConfig {
	for _, config := range huaweiConfigs {
		hasMatch := false
		for _, configRole := range config.Roles {
			if strings.Contains(poolRole, configRole) {
				hasMatch = true
			}
		}
		if hasMatch {
			return &config
		}
	}
	return nil
}

// DeleteNodes terminates huawei ecs instances that have been created.
func DeleteNodes(client *rancher.Client, nodes []*nodes.Node) error {
	hwClient, err := client.GetHuaweiClient()
	if err != nil {
		return err
	}
	request := &ecs_model.DeleteServersRequest{
		Body: &ecs_model.DeleteServersRequestBody{
			DeletePublicip: utils.Pointer(true),
			DeleteVolume:   utils.Pointer(true),
			Servers:        nil,
		},
	}
	for _, node := range nodes {
		request.Body.Servers = append(request.Body.Servers, ecs_model.ServerId{
			Id: node.NodeID,
		})
		logrus.Infof("request to delete huawei ecs instance [%v]", node.NodeID)
	}
	_, err = hwClient.Client.DeleteServers(request)
	return err
}
