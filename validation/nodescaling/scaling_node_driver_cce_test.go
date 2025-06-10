//go:build (validation || infra.cce || extended) && !infra.any && !infra.aks && !infra.gke && !infra.rke2k3s && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package nodescaling

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/cce"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/scalinginput"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CCENodeScalingTestSuite struct {
	suite.Suite
	client        *rancher.Client
	session       *session.Session
	scalingConfig *scalinginput.Config
}

func (s *CCENodeScalingTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *CCENodeScalingTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	s.scalingConfig = new(scalinginput.Config)
	config.LoadConfig(scalinginput.ConfigurationFileKey, s.scalingConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(s.T(), err)

	s.client = client
}

func (s *CCENodeScalingTestSuite) TestScalingCCENodePools() {
	scaleOneNode := cce.NodePool{
		InitialNodeCount: oneNode,
	}

	scaleTwoNodes := cce.NodePool{
		InitialNodeCount: twoNodes,
	}

	tests := []struct {
		name     string
		cceNodes cce.NodePool
		client   *rancher.Client
	}{
		{"Scaling node group by 1", scaleOneNode, s.client},
		{"Scaling node group by 2", scaleTwoNodes, s.client},
	}

	for _, tt := range tests {
		clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
		require.NoError(s.T(), err)

		s.Run(tt.name, func() {
			scalingCCENodePools(s.T(), s.client, clusterID, &tt.cceNodes)
		})
	}
}

func (s *CCENodeScalingTestSuite) TestScalingCCENodePoolsDynamicInput() {
	if s.scalingConfig.CCENodePool == nil {
		s.T().Skip()
	}

	clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
	require.NoError(s.T(), err)

	scalingCCENodePools(s.T(), s.client, clusterID, s.scalingConfig.CCENodePool)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCCENodeScalingTestSuite(t *testing.T) {
	suite.Run(t, new(CCENodeScalingTestSuite))
}
