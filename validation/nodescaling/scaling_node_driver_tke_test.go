//go:build (validation || infra.tke || extended) && !infra.any && !infra.aks && !infra.gke && !infra.rke2k3s && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !sanity && !stress

package nodescaling

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/clusters/tke"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/scalinginput"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TKENodeScalingTestSuite struct {
	suite.Suite
	client        *rancher.Client
	session       *session.Session
	scalingConfig *scalinginput.Config
}

func (s *TKENodeScalingTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *TKENodeScalingTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	s.scalingConfig = new(scalinginput.Config)
	config.LoadConfig(scalinginput.ConfigurationFileKey, s.scalingConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(s.T(), err)

	s.client = client
}

func (s *TKENodeScalingTestSuite) TestScalingTKENodePools() {
	scaleOneNode := tke.NodePoolDetail{
		AutoScalingGroupPara: tke.AutoScalingGroupPara{
			DesiredCapacity: oneNode,
		},
	}

	scaleTwoNodes := tke.NodePoolDetail{
		AutoScalingGroupPara: tke.AutoScalingGroupPara{
			DesiredCapacity: twoNodes,
		},
	}

	tests := []struct {
		name     string
		tkeNodes tke.NodePoolDetail
		client   *rancher.Client
	}{
		{"Scaling node group by 1", scaleOneNode, s.client},
		{"Scaling node group by 2", scaleTwoNodes, s.client},
	}

	for _, tt := range tests {
		clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
		require.NoError(s.T(), err)

		s.Run(tt.name, func() {
			scalingTKENodePools(s.T(), s.client, clusterID, &tt.tkeNodes)
		})
	}
}

func (s *TKENodeScalingTestSuite) TestScalingTKENodePoolsDynamicInput() {
	if s.scalingConfig.TKENodePool == nil {
		s.T().Skip()
	}

	clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
	require.NoError(s.T(), err)

	scalingTKENodePools(s.T(), s.client, clusterID, s.scalingConfig.TKENodePool)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestTKENodeScalingTestSuite(t *testing.T) {
	suite.Run(t, new(TKENodeScalingTestSuite))
}
