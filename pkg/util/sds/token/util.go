package sdstoken

import (
	"github.com/juju/loggo"
	"github.com/samsung-cnct/cma-operator/pkg/util"
)

var (
	logger loggo.Logger
)

func SetLogger() {
	logger = util.GetModuleLogger("pkg.util.sdstoken", loggo.INFO)
}
