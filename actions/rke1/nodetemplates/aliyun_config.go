package nodetemplates

// The json/yaml config key for the Aliyun node template config
const AliyunECSNodeTemplateConfigurationFileKey = "aliyunNodeTemplate"

// AliyunECSNodeTemplateConfig is configuration need to create a Aliyun node template
type AliyunECSNodeTemplateConfig struct {
	AccessKey               string   `json:"accessKeyId" yaml:"accessKeyId"`
	SecretKey               string   `json:"accessKeySecret" yaml:"accessKeySecret"`
	Region                  string   `json:"region" yaml:"region"`
	ImageID                 string   `json:"imageId" yaml:"imageId"`
	SSHPassword             string   `json:"sshPassword" yaml:"sshPassword"`
	SSHKeyPairName          string   `json:"sshKeypair" yaml:"sshKeypair"`
	SSHPrivateKeyPath       string   `json:"sshPrivateKeyPath" yaml:"sshPrivateKeyPath"`
	InstanceType            string   `json:"instanceType" yaml:"instanceType"`
	SecurityGroupID         string   `json:"securityGroupId" yaml:"securityGroupId"`
	SecurityGroupName       string   `json:"securityGroup" yaml:"securityGroup"`
	VpcID                   string   `json:"vpcId" yaml:"vpcId"`
	VSwitchID               string   `json:"vswitchId" yaml:"vswitchId"`
	Zone                    string   `json:"zone" yaml:"zone"`
	PrivateIPOnly           bool     `json:"privateAddressOnly" yaml:"privateAddressOnly"`
	InternetMaxBandwidthOut string   `json:"internetMaxBandwidth" yaml:"internetMaxBandwidth"`
	InternetChargeType      string   `json:"internetChargeType" yaml:"internetChargeType"`
	RouteCIDR               string   `json:"routeCidr" yaml:"routeCidr"`
	SLBID                   string   `json:"slbId" yaml:"slbId"`
	DiskSize                string   `json:"diskSize" yaml:"diskSize"`
	DiskFS                  string   `json:"diskFs" yaml:"diskFs"`
	DiskCategory            string   `json:"diskCategory" yaml:"diskCategory"`
	Description             string   `json:"description" yaml:"description"`
	APIEndpoint             string   `json:"apiEndpoint" yaml:"apiEndpoint"`
	SystemDiskCategory      string   `json:"systemDiskCategory" yaml:"systemDiskCategory"`
	SystemDiskSize          string   `json:"SystemDiskSize" yaml:"SystemDiskSize"`
	ResourceGroupID         string   `json:"resourceGroupId" yaml:"resourceGroupId"`
	InstanceChargeType      string   `json:"instanceChargeType" yaml:"instanceChargeType"`
	Period                  string   `json:"period" yaml:"period"`
	PeriodUnit              string   `json:"periodUnit" yaml:"periodUnit"`
	SpotStrategy            string   `json:"spotStrategy" yaml:"spotStrategy"`
	SpotPriceLimit          string   `json:"spotPriceLimit" yaml:"spotPriceLimit"`
	SpotDuration            string   `json:"spotDuration" yaml:"spotDuration"`
	OpenPorts               []string `json:"openPort" yaml:"openPort"`
}
