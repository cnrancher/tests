//go:build (validation || sanity) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !extended && !stress

package k3s

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
	"github.com/stretchr/testify/require"
)

func (c *CustomClusterProvisioningTestSuite) TestProvisioningK3SCustomClusterDynamicInputStandardUser() {
	isWindows := false
	for _, pool := range c.provisioningConfig.MachinePools {
		if pool.MachinePoolConfig.Windows {
			isWindows = true
			break
		}
	}
	require.False(c.T(), isWindows, "Windows nodes are not supported for k3s Clusters")
	if len(c.provisioningConfig.MachinePools) == 0 {
		c.T().Skip()
	}

	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{provisioninginput.StandardClientName.String(), c.standardUserClient},
	}

	for _, tt := range tests {
		permutations.RunTestPermutations(&c.Suite, tt.name, tt.client, c.provisioningConfig, permutations.K3SCustomCluster, nil, nil)
	}
}
