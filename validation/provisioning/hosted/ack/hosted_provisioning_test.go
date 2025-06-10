//go:build validation

package provisioning

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/cloudcredentials"
	"github.com/rancher/shepherd/extensions/cloudcredentials/ecs"
	"github.com/rancher/shepherd/extensions/clusters/ack"
	"github.com/rancher/shepherd/extensions/users"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/environmentflag"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/pipeline"
	"github.com/rancher/tests/actions/provisioning"
	provisioningInput "github.com/rancher/tests/actions/provisioninginput"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HostedACKClusterProvisioningTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	standardUserClient *rancher.Client
	cluster            *management.Cluster
}

func (h *HostedACKClusterProvisioningTestSuite) TearDownSuite() {
	h.session.Cleanup()
}

func (h *HostedACKClusterProvisioningTestSuite) SetupSuite() {
	testSession := session.NewSession()
	h.session = testSession
	client, err := rancher.NewClient("", testSession)
	require.NoError(h.T(), err)
	h.client = client
	enabled := true
	var testuser = namegen.AppendRandomString("testuser-")
	var testpassword = password.GenerateUserPassword("testpass-")
	user := &management.User{
		Username: testuser,
		Password: testpassword,
		Name:     testuser,
		Enabled:  &enabled,
	}
	newUser, err := users.CreateUserWithRole(client, user, "user")
	require.NoError(h.T(), err)
	newUser.Password = user.Password
	standardUserClient, err := client.AsUser(newUser)
	require.NoError(h.T(), err)
	h.standardUserClient = standardUserClient
}

func (h *HostedACKClusterProvisioningTestSuite) TestProvisioningHostedACK() {
	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{provisioningInput.StandardClientName.String(), h.standardUserClient},
	}
	for _, tt := range tests {
		var ackClusterConfig ack.ClusterConfig
		config.LoadConfig(ack.ACKClusterConfigConfigurationFileKey, &ackClusterConfig)

		clusterObject, err := createProvisioningACKHostedCluster(tt.client, ackClusterConfig)
		require.NoError(h.T(), err)

		provisioning.VerifyHostedCluster(h.T(), tt.client, clusterObject)
	}
}

func createProvisioningACKHostedCluster(client *rancher.Client, ackClusterConfig ack.ClusterConfig) (*management.Cluster, error) {

	cloudCredentialConfig := cloudcredentials.LoadCloudCredential(provisioningInput.AliyunProviderName.String())
	cloudCredential, err := ecs.CreateECSCloudCredentials(client, cloudCredentialConfig)
	if err != nil {
		return nil, err
	}

	clusterName := namegen.AppendRandomString("ackhostcluster")
	clusterResp, err := ack.CreateACKHostedCluster(client, clusterName, cloudCredential.ID, ackClusterConfig, false, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	if client.Flags.GetValue(environmentflag.UpdateClusterName) {
		pipeline.UpdateConfigClusterName(clusterName)
	}

	client, err = client.ReLogin()
	if err != nil {
		return nil, err
	}

	return client.Management.Cluster.ByID(clusterResp.ID)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestHostedACKClusterProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(HostedACKClusterProvisioningTestSuite))
}
