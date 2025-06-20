package aliyun

import (
	srand "crypto/rand"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/pkg/errors"
	"github.com/rancher/machine/libmachine/mcnutils"
	rancherAliyun "github.com/rancher/shepherd/clients/aliyun"
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/pkg/nodes"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	running                = "Running"
	stopped                = "Stopped"
	eipStatusInUse         = "InUse"
	eipStatusAvailable     = "Available"
	https                  = "https"
	defaultTimeout         = 60
	defaultWaitForInterval = 5
	instanceDefaultTimeout = 120
	dockerPort             = 2376
	ipRange                = "0.0.0.0/0"
	defaultSSHUser         = "root"
	dictionary             = "_0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type IpProtocol string

const (
	IpProtocolAll  = IpProtocol("all")
	IpProtocolTCP  = IpProtocol("tcp")
	IpProtocolUDP  = IpProtocol("udp")
	IpProtocolICMP = IpProtocol("icmp")
)

var backoff = wait.Backoff{
	Duration: 30 * time.Second,
	Steps:    20, // 10min
}

type IpPermission struct {
	IpProtocol IpProtocol
	FromPort   int
	ToPort     int
	IpRange    string
}

type Client struct {
	Client *ecs.Client
	Config *rancherAliyun.ECSConfig
}

func CreateNodes(client *rancher.Client, rolesPerPool []string, quantityPerPool []int32) ([]*nodes.Node, error) {
	ecsClient, err := client.GetAliYunClient()
	if err != nil {
		return nil, err
	}
	ecsNodes := []*nodes.Node{}
	for i := len(quantityPerPool) - 1; i >= 0; i-- {
		config := MatchRoleToConfig(rolesPerPool[i], ecsClient.ClientConfig.ECSConfig)
		if config == nil {
			return nil, errors.New("No matching nodesAndRole for Aliyun EcsConfig with role:" + rolesPerPool[i])
		}
		c := &Client{
			Client: ecsClient.Client,
			Config: config,
		}
		request := ecs.CreateCreateInstanceRequest()
		request.Scheme = https
		request.RegionId = ecsClient.ClientConfig.Region
		request.InstanceName = ""
		request.Description = config.Description
		request.ImageId = config.ImageID
		request.InstanceType = config.InstanceType
		request.InternetChargeType = config.InternetChargeType
		request.Password = config.SSHPassword
		request.KeyPairName = config.SSHKeyPairName
		request.VSwitchId = config.VSwitchId
		request.ZoneId = config.Zone
		request.InstanceChargeType = config.InstanceChargeType
		request.ClientToken = createRandomString()
		period, err := strconv.Atoi(config.Period)
		if err != nil {
			return nil, err
		}
		spotDuration, err := strconv.Atoi(config.SpotDuration)
		if err != nil {
			return nil, err
		}
		systemDiskSize, err := strconv.Atoi(config.SystemDiskSize)
		if err != nil {
			return nil, err
		}
		diskSize, err := strconv.Atoi(config.DiskSize)
		if err != nil {
			return nil, err
		}
		request.Period = requests.NewInteger(period)
		request.PeriodUnit = config.PeriodUnit
		request.SpotStrategy = config.SpotStrategy
		var spotPriceLimit float64
		if config.InstanceChargeType == "PostPaid" && config.SpotStrategy == "SpotWithPriceLimit" && config.SpotPriceLimit != "" {
			spotPriceLimit, err = strconv.ParseFloat(config.SpotPriceLimit, 64)
			if err != nil {
				return nil, err
			}
		}
		request.SpotPriceLimit = requests.NewFloat(spotPriceLimit)
		request.SpotDuration = requests.NewInteger(spotDuration)
		if config.SystemDiskCategory != "" {
			request.SystemDiskCategory = config.SystemDiskCategory
		}
		if systemDiskSize > 0 {
			request.SystemDiskSize = requests.NewInteger(systemDiskSize)
		}
		if diskSize > 0 {
			disk := ecs.CreateInstanceDataDisk{
				DiskName:           "aliyun_data",
				Description:        "Data volume for Docker",
				Size:               string(diskSize),
				Category:           config.DiskCategory,
				Device:             "/dev/xvdb",
				DeleteWithInstance: "true",
			}
			request.DataDisk = &[]ecs.CreateInstanceDataDisk{disk}
		}
		if err := c.configureSecurityGroup(config.VpcId, config.SecurityGroupName); err != nil {
			return nil, err
		}
		request.SecurityGroupId = config.SecurityGroupId
		response, err := c.Client.CreateInstance(request)
		if err != nil {
			return nil, err
		}
		if err = c.waitForInstance(response.InstanceId, stopped, 300); err != nil {
			return nil, err
		}
		if err = c.configNetwork(config, request.RegionId, response.InstanceId); err != nil {
			return nil, err
		}
		startInstanceRequest := ecs.CreateStartInstanceRequest()
		startInstanceRequest.Scheme = https
		startInstanceRequest.InstanceId = response.InstanceId
		_, err = c.Client.StartInstance(startInstanceRequest)
		if err = c.waitForInstance(response.InstanceId, running, 300); err != nil {
			return nil, err
		}
		describeInstanceAttributeRequest := ecs.CreateDescribeInstanceAttributeRequest()
		describeInstanceAttributeRequest.Scheme = https
		describeInstanceAttributeRequest.InstanceId = response.InstanceId
		instance, err := c.Client.DescribeInstanceAttribute(describeInstanceAttributeRequest)
		if err != nil {
			return nil, err
		}
		// Sometimes ECS is in a normal "running" state,
		// but during system initialization it requires some time to complete.
		// Therefore, a 1-minute waiting time has been added here.
		time.Sleep(60 * time.Second)
		sshKey, err := nodes.GetSSHKey(config.SSHKeyPairName)
		if err != nil {
			return nil, err
		}
		if config.AliyunUser == "" {
			config.AliyunUser = defaultSSHUser
		}
		ecsNode := &nodes.Node{
			NodeID:           response.InstanceId,
			PublicIPAddress:  c.getIP(instance),
			PrivateIPAddress: c.getPrivateIP(instance),
			SSHUser:          config.AliyunUser,
			SSHKey:           sshKey,
		}
		ecsNodes = append(ecsNodes, ecsNode)
	}
	client.Session.RegisterCleanupFunc(func() error {
		return DeleteNodes(client, ecsNodes)
	})
	return ecsNodes, nil
}

// DeleteNodes terminates ecs instances that have been created.
func DeleteNodes(client *rancher.Client, nodes []*nodes.Node) error {
	ecsClient, err := client.GetAliYunClient()
	if err != nil {
		return err
	}
	c := &Client{
		Client: ecsClient.Client,
	}
	var instanceIDs []string
	for _, node := range nodes {
		if err := c.removeEip(node.NodeID); err != nil {
			logrus.Infof("Delete instances %s eip error %v", node.NodeID, err)
		}
		instanceIDs = append(instanceIDs, node.NodeID)
	}
	if err := c.stopInstances(instanceIDs); err != nil {
		logrus.Errorf("Delete instances error %v", err)
		return err
	}
	request := ecs.CreateDeleteInstancesRequest()
	request.Scheme = https
	request.InstanceId = &instanceIDs
	_, err = ecsClient.Client.DeleteInstances(request)
	if err != nil {
		logrus.Infof("Delete instances error %v", err)
		return err
	}
	return nil
}

func (c *Client) getIP(inst *ecs.DescribeInstanceAttributeResponse) string {
	if inst.PublicIpAddress.IpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 {
		return inst.PublicIpAddress.IpAddress[0]
	}
	if len(inst.EipAddress.IpAddress) > 0 {
		return inst.EipAddress.IpAddress
	}
	return ""
}

func (c *Client) getPrivateIP(instance *ecs.DescribeInstanceAttributeResponse) string {
	if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		return instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}
	return ""
}

func (c *Client) waitForInstance(instanceID string, status string, timeout int) error {
	if timeout <= 0 {
		timeout = instanceDefaultTimeout
	}
	for {
		request := ecs.CreateDescribeInstanceAttributeRequest()
		request.Scheme = https
		request.InstanceId = instanceID
		instance, err := c.Client.DescribeInstanceAttribute(request)
		if err != nil {
			return err
		}
		if instance.Status == status {
			//TODO
			//Sleep one more time for timing issues
			time.Sleep(defaultWaitForInterval * time.Second)
			break
		}
		timeout = timeout - defaultWaitForInterval
		if timeout <= 0 {
			return errors.New("time out")
		}
		time.Sleep(defaultWaitForInterval * time.Second)
	}
	return nil
}

func (c *Client) configNetwork(config *rancherAliyun.ECSConfig, regionID, instanceId string) error {
	var err error
	if config.VpcId != "" {
		// Create EIP for virtual private cloud
		request := ecs.CreateAllocateEipAddressRequest()
		request.Scheme = https
		request.RegionId = regionID
		request.Bandwidth = config.InternetMaxBandwidthOut
		request.InternetChargeType = config.InternetChargeType
		request.ClientToken = createRandomString()
		logrus.Infof("Allocating Eip address for instance %s ...", instanceId)
		response, err := c.Client.AllocateEipAddress(request)
		if err != nil {
			return fmt.Errorf("Failed to allocate EIP address: %v", err)
		}
		if err := c.waitForEip(regionID, response.AllocationId, eipStatusAvailable, 60); err != nil {
			logrus.Infof("Releasing Eip address %s for ...", response.AllocationId)
			releaseEipRequest := ecs.CreateReleaseEipAddressRequest()
			releaseEipRequest.Scheme = https
			releaseEipRequest.AllocationId = response.AllocationId
			_, err2 := c.Client.ReleaseEipAddress(releaseEipRequest)
			if err2 != nil {
				return fmt.Errorf("Failed to release EIP address: %v", err2)
			}
		}
		logrus.Infof("Associating Eip address %s for instance %s ...", response.AllocationId, instanceId)
		createAssociateEipRequest := ecs.CreateAssociateEipAddressRequest()
		createAssociateEipRequest.Scheme = https
		createAssociateEipRequest.AllocationId = response.AllocationId
		createAssociateEipRequest.InstanceId = instanceId
		_, err = c.Client.AssociateEipAddress(createAssociateEipRequest)
		if err != nil {
			return fmt.Errorf("Failed to associate EIP address: %v", err)
		}
		if err := c.waitForEip(regionID, response.AllocationId, eipStatusInUse, 60); err != nil {
			return fmt.Errorf("Failed to wait EIP %s: %v", response.AllocationId, err)
		}
	}
	return err
}

func (c *Client) waitForEip(regionID string, allocationId string, status string, timeout int) error {
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	request := ecs.CreateDescribeEipAddressesRequest()
	request.Scheme = https
	request.RegionId = regionID
	request.AllocationId = allocationId
	for {
		eips, err := c.Client.DescribeEipAddresses(request)
		if err != nil {
			return err
		}
		eipAddress := eips.EipAddresses.EipAddress
		if len(eipAddress) == 0 {
			return errors.New("not found")
		}
		if eipAddress[0].Status == status {
			break
		}
		timeout = timeout - defaultWaitForInterval
		if timeout <= 0 {
			return errors.New("time out")
		}
		time.Sleep(defaultWaitForInterval * time.Second)
	}
	return nil
}

func (c *Client) stopInstances(instanceIDs []string) error {
	request := ecs.CreateStopInstancesRequest()
	request.InstanceId = &instanceIDs
	request.Scheme = https
	_, err := c.Client.StopInstances(request)
	if err != nil {
		return err
	}
	for _, instanceID := range instanceIDs {
		if err = c.waitForInstance(instanceID, stopped, 300); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) configureSecurityGroup(vpcID string, groupName string) error {
	var securityGroup *ecs.DescribeSecurityGroupAttributeResponse
	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.Scheme = "https"
	request.SecurityGroupName = groupName
	request.VpcId = vpcID
	newSecurityGroup := false
	for {
		response, err := c.Client.DescribeSecurityGroups(request)
		if err != nil {
			return err
		}
		pageNumber := response.PageNumber
		pageSize := response.PageSize
		TotalCount := response.TotalCount
		for _, grp := range response.SecurityGroups.SecurityGroup {
			if grp.SecurityGroupName == groupName && grp.VpcId == vpcID {
				securityGroup, _ = c.getSecurityGroup(grp.SecurityGroupId)
				break
			}
		}
		if securityGroup != nil {
			break
		}
		if pageNumber*pageSize >= TotalCount {
			break
		}
		request.PageSize = requests.Integer(pageSize)
		request.PageNumber = requests.Integer(pageNumber + 1)
	}
	// if not found, create
	if securityGroup == nil {
		groupID, err := c.createSecurityGroup(vpcID, groupName, "Rancher Machine")
		if err != nil {
			return err
		}
		newSecurityGroup = true
		if err := mcnutils.WaitFor(c.securityGroupAvailableFunc(groupID)); err != nil {
			return err
		}
		securityGroup, err = c.getSecurityGroup(groupID)
		if err != nil {
			return err
		}
	}
	c.Config.SecurityGroupId = securityGroup.SecurityGroupId
	if newSecurityGroup {
		perms := c.configureSecurityGroupPermissions(securityGroup)
		for _, permission := range perms {
			createAuthorizeSecurityGroupRequest := ecs.CreateAuthorizeSecurityGroupRequest()
			createAuthorizeSecurityGroupRequest.SecurityGroupId = securityGroup.SecurityGroupId
			createAuthorizeSecurityGroupRequest.Permissions = &[]ecs.AuthorizeSecurityGroupPermissions{
				{
					IpProtocol:   string(permission.IpProtocol),
					SourceCidrIp: permission.IpRange,
					PortRange:    fmt.Sprintf("%d/%d", permission.FromPort, permission.ToPort),
				},
			}
			_, err := c.Client.AuthorizeSecurityGroup(createAuthorizeSecurityGroupRequest)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *Client) configureSecurityGroupPermissions(group *ecs.DescribeSecurityGroupAttributeResponse) []IpPermission {
	hasSSHPort := false
	hasDockerPort := false
	for _, p := range group.Permissions.Permission {
		portRange := strings.Split(p.PortRange, "/")
		fromPort, _ := strconv.Atoi(portRange[0])
		switch fromPort {
		case 22:
			hasSSHPort = true
		case dockerPort:
			hasDockerPort = true
		}
	}
	perms := []IpPermission{}
	if !hasSSHPort {
		perms = append(perms, IpPermission{
			IpProtocol: IpProtocolTCP,
			FromPort:   22,
			ToPort:     22,
			IpRange:    ipRange,
		})
	}
	if !hasDockerPort {
		perms = append(perms, IpPermission{
			IpProtocol: IpProtocolTCP,
			FromPort:   dockerPort,
			ToPort:     dockerPort,
			IpRange:    ipRange,
		})
	}
	// If a security group is passed in that needs to be opened, the value passed in is used, if not it is created by default
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolTCP,
		FromPort:   80,
		ToPort:     80,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolTCP,
		FromPort:   443,
		ToPort:     443,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolICMP,
		FromPort:   -1,
		ToPort:     -1,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolTCP,
		FromPort:   6443,
		ToPort:     6443,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolTCP,
		FromPort:   2379,
		ToPort:     2380,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolTCP,
		FromPort:   10250,
		ToPort:     10252,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolTCP,
		FromPort:   10256,
		ToPort:     10256,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolUDP,
		FromPort:   4789,
		ToPort:     4789,
		IpRange:    ipRange,
	})
	perms = append(perms, IpPermission{
		IpProtocol: IpProtocolUDP,
		FromPort:   8472,
		ToPort:     8472,
		IpRange:    ipRange,
	})
	if c.Config.VpcId != "" || c.Config.VSwitchId != "" {
		containerIPRange, err := getContainerCIDR(c.Config.RouteCIDR)
		if err == nil {
			perms = append(perms, IpPermission{
				IpProtocol: IpProtocolAll,
				FromPort:   -1,
				ToPort:     -1,
				IpRange:    containerIPRange,
			})
		}
	}
	return perms
}

func (c *Client) getSecurityGroup(id string) (sg *ecs.DescribeSecurityGroupAttributeResponse, err error) {
	request := ecs.CreateDescribeSecurityGroupAttributeRequest()
	request.Scheme = https
	request.SecurityGroupId = id
	response, err := c.Client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (c *Client) securityGroupAvailableFunc(id string) func() bool {
	return func() bool {
		_, err := c.getSecurityGroup(id)
		if err == nil {
			return true
		}
		return false
	}
}

func (c *Client) createSecurityGroup(vpcID, groupName, description string) (securityGroupID string, err error) {
	request := ecs.CreateCreateSecurityGroupRequest()
	request.Scheme = https
	request.VpcId = vpcID
	request.SecurityGroupName = groupName
	request.Description = description
	response, err := c.Client.CreateSecurityGroup(request)
	if err != nil {
		return "", err
	}
	return response.SecurityGroupId, nil
}

func (c *Client) unassociateEipAddress(instanceID, allocationID string) error {
	request := ecs.CreateUnassociateEipAddressRequest()
	request.Scheme = https
	request.AllocationId = allocationID
	request.InstanceId = instanceID
	_, err := c.Client.UnassociateEipAddress(request)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) releaseEipAddress(allocationID string) error {
	request := ecs.CreateReleaseEipAddressRequest()
	request.Scheme = https
	request.AllocationId = allocationID
	_, err := c.Client.ReleaseEipAddress(request)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) removeEip(instanceID string) error {
	request := ecs.CreateDescribeInstanceAttributeRequest()
	request.Scheme = https
	request.InstanceId = instanceID
	instance, err := c.Client.DescribeInstanceAttribute(request)
	if err != nil {
		return err
	}
	if len(instance.EipAddress.AllocationId) != 0 {
		allocationID := instance.EipAddress.AllocationId
		if err := c.unassociateEipAddress(instanceID, allocationID); err != nil {
			return err
		}
		if err = c.waitForEip(instance.RegionId, allocationID, eipStatusAvailable, 0); err != nil {
			return err
		}
		if err := c.releaseEipAddress(allocationID); err != nil {
			return err
		}
	}
	return nil
}

// CreateRandomString create random string
func createRandomString() string {
	b := make([]byte, 32)
	l := len(dictionary)
	_, err := srand.Read(b)
	if err != nil {
		// fail back to insecure rand
		rand.Seed(time.Now().UnixNano())
		for i := range b {
			b[i] = dictionary[rand.Int()%l]
		}
	} else {
		for i, v := range b {
			b[i] = dictionary[v%byte(l)]
		}
	}
	return string(b)
}

func getContainerCIDR(cidrBlock string) (string, error) {
	ip, _, err := net.ParseCIDR(cidrBlock)
	if err != nil {
		return "", err
	}
	ip = ip.To4()
	ip[2] = 0
	ip[3] = 0
	return fmt.Sprintf("%s/16", ip.String()), nil
}

// MatchRoleToConfig matches the role of nodesAndRoles to the ec2Config that allows this role.
func MatchRoleToConfig(poolRole string, aliyunConfigs []rancherAliyun.ECSConfig) *rancherAliyun.ECSConfig {
	for _, config := range aliyunConfigs {
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
