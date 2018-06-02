package cma

import (
	"github.com/juju/loggo"
	"github.com/samsung-cnct/cma-operator/pkg/generated/cma/client/clientset/versioned"
	"github.com/samsung-cnct/cma-operator/pkg/util"
	"github.com/samsung-cnct/cma-operator/pkg/util/k8sutil"
	"k8s.io/client-go/rest"
)

var (
	logger loggo.Logger
)

func prepareRestClient(config *rest.Config) *versioned.Clientset {
	if config == nil {
		config = k8sutil.DefaultConfig
	}

	return versioned.NewForConfigOrDie(config)
}

func SetLogger() {
	logger = util.GetModuleLogger("pkg.util.ccutil", loggo.INFO)
}
