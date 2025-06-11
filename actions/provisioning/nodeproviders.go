package provisioning

import (
	"fmt"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/nodes"
	"github.com/rancher/tests/actions/nodes/aliyun"
	"github.com/rancher/tests/actions/nodes/cvm"
	"github.com/rancher/tests/actions/nodes/ec2"
	"github.com/rancher/tests/actions/nodes/huawei"
)

const (
	ec2NodeProviderName    = "ec2"
	cvmNodeProviderName    = "cvm"
	huaweiNodePorviderName = "huawei"
	fromConfig             = "config"
	aliyunPorviderName     = "aliyun"
)

type NodeCreationFunc func(client *rancher.Client, rolesPerPool []string, quantityPerPool []int32) (nodes []*nodes.Node, err error)
type NodeDeletionFunc func(client *rancher.Client, nodes []*nodes.Node) error

type ExternalNodeProvider struct {
	Name             string
	NodeCreationFunc NodeCreationFunc
	NodeDeletionFunc NodeDeletionFunc
}

// ExternalNodeProviderSetup is a helper function that setups an ExternalNodeProvider object is a wrapper
// for the specific outside node provider node creator function
func ExternalNodeProviderSetup(providerType string) ExternalNodeProvider {
	switch providerType {
	case ec2NodeProviderName:
		return ExternalNodeProvider{
			Name:             providerType,
			NodeCreationFunc: ec2.CreateNodes,
			NodeDeletionFunc: ec2.DeleteNodes,
		}
	case cvmNodeProviderName:
		return ExternalNodeProvider{
			Name:             providerType,
			NodeCreationFunc: cvm.CreateNodes,
			NodeDeletionFunc: cvm.DeleteNodes,
		}
	case huaweiNodePorviderName:
		return ExternalNodeProvider{
			Name:             providerType,
			NodeCreationFunc: huawei.CreateNodes,
			NodeDeletionFunc: huawei.DeleteNodes,
		}
	case aliyunPorviderName:
		return ExternalNodeProvider{
			Name:             providerType,
			NodeCreationFunc: aliyun.CreateNodes,
			NodeDeletionFunc: aliyun.DeleteNodes,
		}
	case fromConfig:
		return ExternalNodeProvider{
			Name: providerType,
			NodeCreationFunc: func(client *rancher.Client, rolesPerPool []string, quantityPerPool []int32) (nodesList []*nodes.Node, err error) {
				var nodeConfig nodes.ExternalNodeConfig
				config.LoadConfig(nodes.ExternalNodeConfigConfigurationFileKey, &nodeConfig)

				nodesList = nodeConfig.Nodes[-1]

				for _, node := range nodesList {
					sshKey, err := nodes.GetSSHKey(node.SSHKeyName)
					if err != nil {
						return nil, err
					}

					node.SSHKey = sshKey
				}
				return nodesList, nil
			},
			NodeDeletionFunc: func(client *rancher.Client, nodes []*nodes.Node) error {
				return ec2.DeleteNodes(client, nodes)
			},
		}
	default:
		panic(fmt.Sprintf("Node Provider:%v not found", providerType))
	}

}
