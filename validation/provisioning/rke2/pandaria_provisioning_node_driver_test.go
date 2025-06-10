//go:build (validation || extended) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke2k3s && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package rke2

import (
	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/tests/actions/provisioning/permutations"
	"github.com/rancher/tests/actions/provisioninginput"
)

func (r *RKE2NodeDriverProvisioningTestSuite) TestPandariaProvisioningRKE2Cluster() {
	nodeRoles0 := []provisioninginput.MachinePools{provisioninginput.EtcdMachinePool, provisioninginput.ControlPlaneMachinePool, provisioninginput.WorkerMachinePool}

	tests := []struct {
		name         string
		machinePools []provisioninginput.MachinePools
		client       *rancher.Client
	}{
		{"3 nodes - 1 role per node Standard User", nodeRoles0, r.standardUserClient},
	}

	for _, tt := range tests {
		provisioningConfig := *r.provisioningConfig
		provisioningConfig.MachinePools = tt.machinePools
		permutations.RunTestPermutations(&r.Suite, tt.name, tt.client, &provisioningConfig, permutations.RKE2ProvisionCluster, nil, nil)
	}
}
