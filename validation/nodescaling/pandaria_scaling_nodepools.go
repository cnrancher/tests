package nodescaling

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters/ack"
	"github.com/rancher/shepherd/extensions/clusters/cce"
	"github.com/rancher/shepherd/extensions/clusters/tke"
	"github.com/stretchr/testify/require"
)

func scalingCCENodePools(t *testing.T, client *rancher.Client, clusterID string, nodePool *cce.NodePool) {
	cluster, err := client.Management.Cluster.ByID(clusterID)
	require.NoError(t, err)

	clusterResp, err := cce.ScalingCCENodePoolsNodes(client, cluster, nodePool)
	require.NoError(t, err)

	nodePool.InitialNodeCount = -nodePool.InitialNodeCount
	_, err = cce.ScalingCCENodePoolsNodes(client, clusterResp, nodePool)
	require.NoError(t, err)
}

func scalingACKNodePools(t *testing.T, client *rancher.Client, clusterID string, nodePool *ack.NodePoolInfo) {
	cluster, err := client.Management.Cluster.ByID(clusterID)
	require.NoError(t, err)

	clusterResp, err := ack.ScalingACKNodePoolsNodes(client, cluster, nodePool)
	require.NoError(t, err)

	nodePool.InstancesNum = -nodePool.InstancesNum
	_, err = ack.ScalingACKNodePoolsNodes(client, clusterResp, nodePool)
	require.NoError(t, err)
}

func scalingTKENodePools(t *testing.T, client *rancher.Client, clusterID string, nodePool *tke.NodePoolDetail) {
	cluster, err := client.Management.Cluster.ByID(clusterID)
	require.NoError(t, err)

	clusterResp, err := tke.ScalingTKENodePoolsNodes(client, cluster, nodePool)
	require.NoError(t, err)

	nodePool.AutoScalingGroupPara.DesiredCapacity = -nodePool.AutoScalingGroupPara.DesiredCapacity
	_, err = tke.ScalingTKENodePoolsNodes(client, clusterResp, nodePool)
	require.NoError(t, err)
}
