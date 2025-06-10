//go:build (validation || extended) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package rke1

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
)

func (r *RKE1NodeDriverProvisioningTestSuite) TestPandariaProvisioningRKE1Cluster() {
	if len(r.provisioningConfig.NodePools) == 0 {
		r.T().Skip()
	}

	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{provisioninginput.StandardClientName.String(), r.standardUserClient},
	}
	for _, tt := range tests {
		permutations.RunTestPermutations(&r.Suite, tt.name, tt.client, r.provisioningConfig, permutations.RKE1ProvisionCluster, nil, nil)
	}
}
