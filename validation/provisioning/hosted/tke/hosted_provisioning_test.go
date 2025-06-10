//go:build validation

package tke

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/users"
	password "github.com/rancher/shepherd/extensions/users/passwordgenerator"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/provisioning"
	provisioningInput "github.com/rancher/tests/actions/provisioninginput"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HostedTKEClusterProvisioningTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	standardUserClient *rancher.Client
}

func (h *HostedTKEClusterProvisioningTestSuite) TearDownSuite() {
	h.session.Cleanup()
}

func (h *HostedTKEClusterProvisioningTestSuite) SetupSuite() {
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

func (h *HostedTKEClusterProvisioningTestSuite) TestProvisioningHostedTKE() {
	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{provisioningInput.AdminClientName.String(), h.client},
	}

	for _, tt := range tests {
		clusterObject, err := provisioning.CreateProvisioningTKEHostedCluster(tt.client)
		require.NoError(h.T(), err)

		provisioning.VerifyHostedCluster(h.T(), tt.client, clusterObject)
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestHostedTKEClusterProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(HostedTKEClusterProvisioningTestSuite))
}
