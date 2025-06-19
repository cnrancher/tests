//go:build (validation || extended) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package k3s

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tests/actions/provisioning/permutations"
	provisioningInput "github.com/rancher/tests/actions/provisioninginput"
)

func (k *K3SNodeDriverProvisioningTestSuite) TestProvisioningK3SClusterStandardUser() {
	nodeRoles0 := []provisioningInput.MachinePools{provisioningInput.AllRolesMachinePool}

	tests := []struct {
		name         string
		machinePools []provisioningInput.MachinePools
		client       *rancher.Client
	}{
		{"3 nodes - 1 role per node Standard User", nodeRoles0, k.standardUserClient},
	}

	for _, tt := range tests {
		provisioningConfig := *k.provisioningConfig
		provisioningConfig.MachinePools = tt.machinePools
		permutations.RunTestPermutations(&k.Suite, tt.name, tt.client, &provisioningConfig, permutations.K3SProvisionCluster, nil, nil)
	}
}
