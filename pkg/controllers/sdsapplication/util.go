package sdsapplication

import (
	"github.com/golang/glog"
	"github.com/juju/loggo"
	"github.com/samsung-cnct/cma-operator/pkg/util"
	"github.com/samsung-cnct/cma-operator/pkg/util/cmagrpc"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var (
	logger loggo.Logger
)

func retrieveClusterRestConfig(name string, kubeconfig string) (*rest.Config, error) {
	// Let's create a tempfile and line it up for removal
	file, err := ioutil.TempFile(os.TempDir(), "cma-kubeconfig")
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

func (c *SDSApplicationController) getRestConfigForRemoteCluster(clusterName string, namespace string, config *rest.Config) (*rest.Config, error) {
	cluster, err := c.cmaGRPCClient.GetCluster(cmagrpc.GetClusterInput{Name: clusterName})
	if err != nil {
		glog.Errorf("Failed to retrieve Cluster  -->%s<-- in namespace -->%s<--, error was: %s", clusterName, namespace, err)
		return nil, err
	}
	if cluster.Kubeconfig == "" {
		glog.Errorf("Could not install tiller yet for cluster -->%s<-- cluster is not ready, status is -->%s<--", cluster.Name, cluster.Status)
		return nil, err
	}

	remoteConfig, err := retrieveClusterRestConfig(clusterName, cluster.Kubeconfig)
	if err != nil {
		glog.Errorf("Could not install tiller yet for cluster -->%s<-- cluster is not ready, error is: %v", clusterName, err)
		return nil, err
	}

	return remoteConfig, nil
}

func (c *SDSApplicationController) SetLogger() {
	logger = util.GetModuleLogger("pkg.controllers.sdsapplication", loggo.INFO)
}
