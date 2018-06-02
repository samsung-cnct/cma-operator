package sdsapplication

import (
	"github.com/golang/glog"
	"github.com/juju/loggo"
	"github.com/samsung-cnct/cma-operator/pkg/util"
	"github.com/samsung-cnct/cma-operator/pkg/util/ccutil"
	"io/ioutil"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

var (
	logger loggo.Logger
)

func retrieveClusterRestConfig(name string, namespace string, config *rest.Config) (*rest.Config, error) {
	cluster, err := ccutil.GetKrakenCluster(name, namespace, config)
	if err != nil {
		return nil, err
	}
	// Let's create a tempfile and line it up for removal
	file, err := ioutil.TempFile(os.TempDir(), "kraken-kubeconfig")
	defer os.Remove(file.Name())
	file.WriteString(cluster.Status.Kubeconfig)

	clusterConfig, err := clientcmd.BuildConfigFromFlags("", file.Name())
	if os.Getenv("CLUSTERMANAGERAPI_INSECURE_TLS") == "true" {
		clusterConfig.TLSClientConfig = rest.TLSClientConfig{Insecure: true}
	}

	if err != nil {
		logger.Errorf("Could not load kubeconfig for cluster -->%s<-- in namespace -->%s<--", name, namespace)
		return nil, err
	}
	return clusterConfig, nil
}

func (c *SDSApplicationController) getRestConfigForRemoteCluster(clusterName string, namespace string, config *rest.Config) (*rest.Config, error) {
	cluster, err := ccutil.GetKrakenCluster(clusterName, namespace, config)
	if err != nil {
		glog.Errorf("Failed to retrieve KrakenCluster CR -->%s<-- in namespace -->%s<--, error was: %s", clusterName, namespace, err)
		return nil, err
	}
	if cluster.Status.Kubeconfig == "" {
		glog.Errorf("Could not install tiller yet for cluster -->%s<-- cluster is not ready, status is -->%s<--", cluster.Name, cluster.Status.State)
		return nil, err
	}

	remoteConfig, err := retrieveClusterRestConfig(clusterName, namespace, config)
	if err != nil {
		glog.Errorf("Could not install tiller yet for cluster -->%s<-- cluster is not ready, error is: %v", clusterName, err)
		return nil, err
	}

	return remoteConfig, nil
}

func (c *SDSApplicationController) SetLogger() {
	logger = util.GetModuleLogger("pkg.controllers.sdsapplication", loggo.INFO)
}
