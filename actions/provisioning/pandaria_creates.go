package provisioning

import (
	"time"

	"github.com/rancher/shepherd/clients/rancher"
	management "github.com/rancher/shepherd/clients/rancher/generated/management/v3"
	"github.com/rancher/shepherd/extensions/cloudcredentials"
	"github.com/rancher/shepherd/extensions/cloudcredentials/huawei"
	"github.com/rancher/shepherd/extensions/cloudcredentials/tencent"
	"github.com/rancher/shepherd/extensions/clusters/cce"
	"github.com/rancher/shepherd/extensions/clusters/tke"
	"github.com/rancher/shepherd/extensions/defaults"
	"github.com/rancher/shepherd/pkg/environmentflag"
	namegen "github.com/rancher/shepherd/pkg/namegenerator"
	"github.com/rancher/tests/actions/pipeline"
	"github.com/sirupsen/logrus"
	kwait "k8s.io/apimachinery/pkg/util/wait"
)

// CreateProvisioningCCEHostedCluster provisions an CCE cluster, then runs verify checks
func CreateProvisioningCCEHostedCluster(client *rancher.Client) (*management.Cluster, error) {
	cloudCredentialConfig := cloudcredentials.LoadCloudCredential("huawei")
	cloudCredential, err := huawei.CreateHuaweiCloudCredentials(client, cloudCredentialConfig)
	if err != nil {
		return nil, err
	}
	clusterName := namegen.AppendRandomString("ccehostcluster")
	clusterResp, err := cce.CreateCCEHostedCluster(client, clusterName, cloudCredential.ID, false, false, false, false, map[string]string{})
	if err != nil {
		return nil, err
	}
	if client.Flags.GetValue(environmentflag.UpdateClusterName) {
		pipeline.UpdateConfigClusterName(clusterName)
	}
	err = kwait.Poll(10*time.Second, defaults.ThirtyMinuteTimeout, func() (bool, error) {
		done, err := cce.UpdateNodePublicIP(client, clusterResp.ID)
		if err != nil {
			logrus.Errorf("UpdateNodePublicIP failed: %v", err)
		}
		return done, err
	})
	if err != nil {
		return nil, err
	}
	client, err = client.ReLogin()
	if err != nil {
		return nil, err
	}
	return client.Management.Cluster.ByID(clusterResp.ID)
}

// CreateProvisioningTKEHostedCluster provisions an TKE cluster, then runs verify checks
func CreateProvisioningTKEHostedCluster(client *rancher.Client) (*management.Cluster, error) {
	cloudCredentialConfig := cloudcredentials.LoadCloudCredential("tke")
	cloudCredential, err := tencent.CreateTencentCloudCredentials(client, cloudCredentialConfig)
	if err != nil {
		return nil, err
	}
	clusterName := namegen.AppendRandomString("tkehostcluster")
	clusterResp, err := tke.CreateTKEHostedCluster(client, clusterName, cloudCredential.ID, false, false, false, false, nil)
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
