package cvm

import (
	"errors"
	"strings"
	"time"

	rancherCvm "github.com/rancher/shepherd/clients/cvm"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/nodes"
	"github.com/rancher/tests/actions/provisioninginput"
	"github.com/sirupsen/logrus"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvmapi "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	nodeBaseName = "rancher-automation"
)

var backoff = wait.Backoff{
	Duration: 30 * time.Second,
	Steps:    12,
}

// CreateNodes creates `quantityPerPool[n]` number of cvm instances
func CreateNodes(client *rancher.Client, rolesPerPool []string, quantityPerPool []int32) (cvmNodes []*nodes.Node, err error) {
	cvmClient, err := client.GetCVMClient()
	if err != nil {
		return nil, err
	}
	runningReservations := []*cvmapi.RunInstancesResponse{}
	reservationConfigs := []*rancherCvm.TencentCVMConfig{}
	for i := len(quantityPerPool) - 1; i >= 0; i-- {
		config := MatchRoleToConfig(rolesPerPool[i], cvmClient.ClientConfig.TencentCVMConfig)
		if config == nil {
			return nil, errors.New("No matching nodesAndRole for TencentCVMConfig with role:" + rolesPerPool[i])
		}
		runInstancesRequest := cvmapi.NewRunInstancesRequest()
		runInstancesRequest.Placement = &cvmapi.Placement{
			Zone: common.StringPtr(config.Zone),
		}
		runInstancesRequest.InstanceType = common.StringPtr(config.InstanceType)
		runInstancesRequest.InternetAccessible = &cvmapi.InternetAccessible{
			InternetChargeType:      common.StringPtr(config.InternetChargeType),
			PublicIpAssigned:        common.BoolPtr(config.PublicIPAssigned),
			InternetMaxBandwidthOut: common.Int64Ptr(int64(config.InternetMaxBandwidthOut)),
		}
		runInstancesRequest.VirtualPrivateCloud = &cvmapi.VirtualPrivateCloud{
			VpcId:    common.StringPtr(config.VpcID),
			SubnetId: common.StringPtr(config.SubnetID),
		}
		runInstancesRequest.InstanceName = common.StringPtr(config.InstanceName)
		runInstancesRequest.ImageId = common.StringPtr(config.ImageID)
		runInstancesRequest.SecurityGroupIds = common.StringPtrs(config.SecurityGroupIDs)
		runInstancesRequest.LoginSettings = &cvmapi.LoginSettings{
			KeyIds: common.StringPtrs(config.KeyIDs),
		}
		runInstancesRequest.InstanceCount = common.Int64Ptr(int64(quantityPerPool[i]))
		runInstancesRequest.UserData = common.StringPtr(config.UserData)
		runInstancesRequest.TagSpecification = []*cvmapi.TagSpecification{
			{
				ResourceType: common.StringPtr("instance"),
				Tags: []*cvmapi.Tag{
					{
						Key:   common.StringPtr("Name"),
						Value: common.StringPtr(nodeBaseName),
					},
				},
			},
		}
		reservation, err := cvmClient.Client.RunInstances(runInstancesRequest)
		if err != nil {
			return nil, err
		}
		runningReservations = append(runningReservations, reservation)
		reservationConfigs = append(reservationConfigs, config)
	}
	for i := 0; i < len(quantityPerPool); i++ {
		listOfInstanceIDs := runningReservations[i].Response.InstanceIdSet
		if err = wait.ExponentialBackoff(backoff, func() (bool, error) {
			getInstancesStatusRequest := cvmapi.NewDescribeInstancesStatusRequest()
			getInstancesStatusRequest.InstanceIds = listOfInstanceIDs
			instancesStatus, err := cvmClient.Client.DescribeInstancesStatus(getInstancesStatusRequest)
			if err != nil {
				return false, err
			}
			for _, instance := range instancesStatus.Response.InstanceStatusSet {
				if *instance.InstanceState != "RUNNING" {
					return false, nil
				}
				logrus.Infof("instance %s is running", *instance.InstanceId)
			}
			return true, nil
		}); err != nil {
			return nil, err
		}
		getInstancesRequest := cvmapi.NewDescribeInstancesRequest()
		getInstancesRequest.InstanceIds = listOfInstanceIDs
		instances, err := cvmClient.Client.DescribeInstances(getInstancesRequest)
		if err != nil {
			return nil, err
		}
		readyInstances := instances.Response.InstanceSet
		sshKey := []byte(reservationConfigs[i].TencentSSHKey)
		for _, readyInstance := range readyInstances {
			cvmNode := &nodes.Node{
				NodeID:           *readyInstance.InstanceId,
				PublicIPAddress:  *readyInstance.PublicIpAddresses[0],
				PrivateIPAddress: *readyInstance.PrivateIpAddresses[0],
				SSHUser:          reservationConfigs[i].TencentUser,
				SSHKey:           sshKey,
			}
			provisioningInputConfig := new(provisioninginput.Config)
			config.LoadConfig(provisioninginput.ConfigurationFileKey, provisioningInputConfig)
			if len(provisioningInputConfig.RKE1KubernetesVersions) > 0 {
				// wait for docker installed by cloud-init
				if err = wait.ExponentialBackoff(backoff, func() (bool, error) {
					_, err := cvmNode.ExecuteCommand("docker ps")
					if err != nil {
						logrus.Infof("wait for node [%v] install docker", cvmNode.NodeID)
						return false, nil
					}
					return true, nil
				}); err != nil {
					return nil, err
				}
			}
			// re-reverse the list so that the order is corrected
			cvmNodes = append([]*nodes.Node{cvmNode}, cvmNodes...)
		}
	}
	time.Sleep(20 * time.Second)
	client.Session.RegisterCleanupFunc(func() error {
		return DeleteNodes(client, cvmNodes)
	})
	return cvmNodes, nil
}

// DeleteNodes terminates cvm instances that have been created.
func DeleteNodes(client *rancher.Client, nodes []*nodes.Node) error {
	cvmClient, err := client.GetCVMClient()
	if err != nil {
		return err
	}
	var instanceIDs []*string
	for _, node := range nodes {
		instanceIDs = append(instanceIDs, common.StringPtr(node.NodeID))
	}
	terminateInstancesRequest := cvmapi.NewTerminateInstancesRequest()
	terminateInstancesRequest.InstanceIds = instanceIDs
	_, err = cvmClient.Client.TerminateInstances(terminateInstancesRequest)
	return err
}

// MatchRoleToConfig matches the role of nodesAndRoles to the cvmConfig that allows this role.
func MatchRoleToConfig(poolRole string, cvmConfigs []rancherCvm.TencentCVMConfig) *rancherCvm.TencentCVMConfig {
	for _, config := range cvmConfigs {
		configRoles := " --" + strings.Join(config.Roles, " --")
		if strings.Contains(configRoles, poolRole) {
			return &config
		}
	}
	return nil
}
