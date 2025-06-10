//go:build (validation || infra.ack || extended) && !infra.any && !infra.aks && !infra.gke && !infra.rke2k3s && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package nodescaling

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/ack"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/scalinginput"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ACKNodeScalingTestSuite struct {
	suite.Suite
	client        *rancher.Client
	session       *session.Session
	scalingConfig *scalinginput.Config
}

func (s *ACKNodeScalingTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *ACKNodeScalingTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	s.scalingConfig = new(scalinginput.Config)
	config.LoadConfig(scalinginput.ConfigurationFileKey, s.scalingConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(s.T(), err)

	s.client = client
}

func (s *ACKNodeScalingTestSuite) TestScalingACKNodePools() {
	scaleOneNode := ack.NodePoolInfo{
		InstancesNum: oneNode,
	}

	tests := []struct {
		name     string
		ackNodes ack.NodePoolInfo
		client   *rancher.Client
	}{
		{"Scaling node group by 1", scaleOneNode, s.client},
	}

	for _, tt := range tests {
		clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
		require.NoError(s.T(), err)

		s.Run(tt.name, func() {
			scalingACKNodePools(s.T(), s.client, clusterID, &tt.ackNodes)
		})
	}
}

func (s *ACKNodeScalingTestSuite) TestScalingACKNodePoolsDynamicInput() {
	if s.scalingConfig.ACKNodePool == nil {
		s.T().Skip()
	}

	clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
	require.NoError(s.T(), err)

	scalingACKNodePools(s.T(), s.client, clusterID, s.scalingConfig.ACKNodePool)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestACKNodeScalingTestSuite(t *testing.T) {
	suite.Run(t, new(ACKNodeScalingTestSuite))
}
