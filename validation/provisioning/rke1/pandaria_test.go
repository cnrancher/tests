//go:build (validation || extended) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package rke1

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
)

func (c *CustomClusterProvisioningTestSuite) TestProvisioningRKE1CustomClusterDynamicInputStandardUser() {
	if len(c.provisioningConfig.NodePools) == 0 {
		c.T().Skip()
	}

	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{provisioninginput.StandardClientName.String(), c.standardUserClient},
	}

	for _, tt := range tests {
		permutations.RunTestPermutations(&c.Suite, tt.name, tt.client, c.provisioningConfig, permutations.RKE1CustomCluster, nil, nil)
	}
}
