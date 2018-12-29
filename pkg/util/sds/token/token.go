package sdstoken

import (
	"github.com/golang/glog"
	"github.com/samsung-cnct/cma-operator/pkg/generated/cma/client/clientset/versioned"
	"github.com/samsung-cnct/cma-operator/pkg/util/cmagrpc"
	"github.com/samsung-cnct/cma-operator/pkg/util/helmutil"
	"github.com/samsung-cnct/cma-operator/pkg/util/k8sutil"
	"github.com/vmware/harbor/src/jobservice/logger"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

const (
	SDSServiceAccountName = "sds-sa"
	SDSClusterRoleName = "sds-sa-cluster-role"
)

type SDSTokenClient struct {
	client        *versioned.Clientset
	cmaGRPCClient cmagrpc.ClientInterface
}

func NewSDSTokenClient(config *rest.Config) (*SDSTokenClient, error) {
	cmaGRPCClient, err := cmagrpc.CreateNewDefaultClient()
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = k8sutil.DefaultConfig
	}
	client := versioned.NewForConfigOrDie(config)

	output := &SDSTokenClient{
		client:        client,
		cmaGRPCClient: cmaGRPCClient,
	}

	return output, nil
}

func (c *SDSTokenClient) CreateSDSToken(clusterName string, namespace string) (bool, error){

	config, err := c.getRestConfigForRemoteCluster(clusterName, namespace, nil)
	if err != nil {
		return false, err
	}

	// create token service account
	_, err = k8sutil.CreateServiceAccount(k8sutil.GenerateServiceAccount(SDSServiceAccountName), namespace, config)
	if err != nil {
		logger.Infof("could not create sds-token for cluster -->%s<--, due to the following error %s", clusterName, err)
	}

	// create cluster admin role
	_, err = k8sutil.CreateClusterRole(helmutil.GenerateClusterAdminRole(SDSClusterRoleName), config)
	if err != nil {
		logger.Infof("could not create ClusterRole for cluster -->%s<--, due to the following error %s", clusterName, err)
	}

	// bind token service account to role
	_, err = k8sutil.CreateClusterRoleBinding(k8sutil.GenerateSingleClusterRolebinding(SDSClusterRoleName, SDSServiceAccountName, namespace, SDSClusterRoleName), config)
	if err != nil {
		logger.Infof("could not create ClusterRoleBinding for cluster -->%s<--, due to the following error %s", clusterName, err)
	}

	// retrieve token secret name from service account
	tokenName, err := k8sutil.GetTokenNameFromServiceAccount(SDSServiceAccountName, namespace, config)
	if err != nil {
		logger.Infof("could not get service account token for cluster -->%s<--, due to the following error %s", clusterName, err)
	}
	// get token data from secret
	secret, err := k8sutil.GetSecret(tokenName, namespace, config)
	if err != nil {
		logger.Infof("could not get secret -->%s<-- for service account -->%s<--, due to the following error %s", tokenName, SDSServiceAccountName, err)
	}

	// save token as secret in cmc
	err = k8sutil.CreateSecret(clusterName + "-" + SDSServiceAccountName + "-token", namespace, "token", secret.Data["token"], nil)
	if err != nil {
		logger.Infof("could not save token -->%s<-- for service account -->%s<--, due to the following error %s", tokenName, SDSServiceAccountName, err)
	}

	logger.Info("created %s token", SDSServiceAccountName)
	return true, nil
}

func (c *SDSTokenClient) getRestConfigForRemoteCluster(clusterName string, namespace string, config *rest.Config) (*rest.Config, error) {
	sdscluster, err := c.client.CmaV1alpha1().SDSClusters(namespace).Get(clusterName, v1.GetOptions{})
	if err != nil {
		glog.Errorf("Failed to retrieve SDSCluster -->%s<--, error was: %s", clusterName, err)
		return nil, err
	}
	cluster, err := c.cmaGRPCClient.GetCluster(cmagrpc.GetClusterInput{Name: clusterName, Provider: sdscluster.Spec.Provider})
	if err != nil {
		glog.Errorf("Failed to retrieve Cluster Status -->%s<--, error was: %s", clusterName, err)
		return nil, err
	}
	if cluster.Kubeconfig == "" {
		glog.Errorf("Could not create sds-token yet for cluster -->%s<-- cluster is not ready, status is -->%s<--", cluster.Name, cluster.Status)
		return nil, err
	}

	remoteConfig, err := retrieveClusterRestConfig(clusterName, cluster.Kubeconfig)
	if err != nil {
		glog.Errorf("Could not create sds-token yet for cluster -->%s<-- cluster is not ready, error is: %v", clusterName, err)
		return nil, err
	}

	return remoteConfig, nil
}

func retrieveClusterRestConfig(name string, kubeconfig string) (*rest.Config, error) {
	// Let's create a tempfile and line it up for removal
	file, err := ioutil.TempFile(os.TempDir(), "cluster-kubeconfig")
	defer os.Remove(file.Name())
	file.WriteString(kubeconfig)

	clusterConfig, err := clientcmd.BuildConfigFromFlags("", file.Name())
	if os.Getenv("CLUSTERMANAGERAPI_INSECURE_TLS") == "true" {
		clusterConfig.TLSClientConfig = rest.TLSClientConfig{Insecure: true}
	}

	if err != nil {
		logger.Errorf("Could not load kubeconfig for cluster -->%s<--", name)
		return nil, err
	}
	return clusterConfig, nil
}
