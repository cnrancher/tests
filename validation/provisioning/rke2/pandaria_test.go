//go:build (validation || sanity) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !extended && !stress

package rke2

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
)

func (c *CustomClusterProvisioningTestSuite) TestProvisioningRKE2CustomClusterDynamicInputStandardUser() {
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
		permutations.RunTestPermutations(&c.Suite, tt.name, tt.client, c.provisioningConfig, permutations.RKE2CustomCluster, nil, nil)
	}
}
