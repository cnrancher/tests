package machinepools

import (
	"github.com/rancher/shepherd/pkg/config"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	ECSKind                              = "aliyunecsConfig"
	ECSPoolType                          = "rke-machine-config.cattle.io.aliyunecsconfig"
	ECSResourceConfig                    = "aliyunecsconfigs"
	ECSMachingConfigConfigurationFileKey = "ecsMachineConfigs"
)

type ECSMachineConfigs struct {
	ECSMachineConfig []ECSMachineConfig `json:"ecsMachineConfig" yaml:"ecsMachineConfig"`
	Region           string             `json:"region" yaml:"region"`
}
type ECSMachineConfig struct {
	Roles
	Region               string `json:"region" yaml:"region"`
	Zone                 string `json:"zone" yaml:"zone"`
	ImageID              string `json:"imageID" yaml:"imageID"`
	InstanceChargeType   string `json:"instanceChargeType" yaml:"instanceChargeType"`
	InstanceType         string `json:"instanceType" yaml:"instanceType"`
	DiskFS               string `json:"diskFs" yaml:"diskFs"`
	DiskSize             string `json:"diskSize" yaml:"diskSize"`
	DiskCategory         string `json:"diskCategory" yaml:"diskCategory"`
	InternetChargeType   string `json:"internetChargeType" yaml:"internetChargeType"`
	InternetMaxBandwidth string `json:"internetMaxBandwidth" yaml:"internetMaxBandwidth"`
	IoOptimized          string `json:"ioOptimized" yaml:"ioOptimized"`
	SecurityGroup        string `json:"securityGroup" yaml:"securityGroup"`
	SystemDiskCategory   string `json:"systemDiskCategory" yaml:"systemDiskCategory"`
	SystemDiskSize       string `json:"systemDiskSize" yaml:"systemDiskSize"`
	UpgradeKernel        bool   `json:"upgradeKernel" yaml:"upgradeKernel"`
	VpcID                string `json:"vpcId" yaml:"vpcId"`
	VswitchID            string `json:"vswitchId" yaml:"vswitchId"`
}

func NewECSMachineConfig(generatedPoolName, namespace string) []unstructured.Unstructured {
	var ecsMachineConfigs ECSMachineConfigs
	config.LoadConfig(ECSMachingConfigConfigurationFileKey, &ecsMachineConfigs)
	var multiConfig []unstructured.Unstructured
	for _, ecsMachineConfig := range ecsMachineConfigs.ECSMachineConfig {
		machineConfig := unstructured.Unstructured{}
		machineConfig.SetAPIVersion("rke-machine-config.cattle.io/v1")
		machineConfig.SetKind(ECSKind)
		machineConfig.SetGenerateName(generatedPoolName)
		machineConfig.SetNamespace(namespace)
		machineConfig.Object["region"] = ecsMachineConfig.Region
		machineConfig.Object["zone"] = ecsMachineConfig.Zone
		machineConfig.Object["type"] = ECSPoolType
		machineConfig.Object["imageId"] = ecsMachineConfig.ImageID
		machineConfig.Object["instanceChargeType"] = ecsMachineConfig.InstanceChargeType
		machineConfig.Object["instanceType"] = ecsMachineConfig.InstanceType
		machineConfig.Object["diskFS"] = ecsMachineConfig.DiskFS
		machineConfig.Object["diskSize"] = ecsMachineConfig.DiskSize
		machineConfig.Object["diskCategory"] = ecsMachineConfig.DiskCategory
		machineConfig.Object["systemDiskCategory"] = ecsMachineConfig.SystemDiskCategory
		machineConfig.Object["systemDiskSize"] = ecsMachineConfig.SystemDiskSize
		machineConfig.Object["internetChargeType"] = ecsMachineConfig.InternetChargeType
		machineConfig.Object["internetMaxBandwidth"] = ecsMachineConfig.InternetMaxBandwidth
		machineConfig.Object["ioOptimized"] = ecsMachineConfig.IoOptimized
		machineConfig.Object["securityGroup"] = ecsMachineConfig.SecurityGroup
		machineConfig.Object["upgradeKernel"] = ecsMachineConfig.UpgradeKernel
		machineConfig.Object["vpcId"] = ecsMachineConfig.VpcID
		machineConfig.Object["vswitchId"] = ecsMachineConfig.VswitchID
		multiConfig = append(multiConfig, machineConfig)
	}
	return multiConfig
}
func GetECSMachineRoles() []Roles {
	var ecsMachineConfigs ECSMachineConfigs
	config.LoadConfig(ECSMachingConfigConfigurationFileKey, &ecsMachineConfigs)
	var allRoles []Roles
	for _, ecsMachineConfig := range ecsMachineConfigs.ECSMachineConfig {
		allRoles = append(allRoles, ecsMachineConfig.Roles)
	}
	return allRoles
}
