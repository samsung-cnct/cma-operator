package cmd

import (
	"sync"

	"github.com/samsung-cnct/cma-operator/pkg/util"
	"github.com/samsung-cnct/cma-operator/pkg/util/k8sutil"
	"github.com/spf13/viper"

	"flag"
	"fmt"
	"github.com/juju/loggo"
	"github.com/samsung-cnct/cma-operator/pkg/controllers/sds-cluster"
	"github.com/samsung-cnct/cma-operator/pkg/controllers/sds-package-manager"
	"github.com/samsung-cnct/cma-operator/pkg/controllers/sdsapplication"
	"github.com/samsung-cnct/cma-operator/pkg/util/cma"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"os"
	"strings"
)

var (
	logger loggo.Logger
	config *rest.Config

	rootCmd = &cobra.Command{
		Use:   "operator",
		Short: "CMA Operator",
		Long:  `The CMA Operator`,
		Run: func(cmd *cobra.Command, args []string) {
			operator()
		},
	}
)

func init() {
	viper.SetEnvPrefix("cmaoperator")
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	rootCmd.Flags().String("kubeconfig", "", "Location of kubeconfig file")
	rootCmd.Flags().String("kubernetes-namespace", "default", "Namespace to operate on")
	rootCmd.Flags().String("cma-endpoint", "", "Location of the Cluster Manager API GRPC service")
	rootCmd.Flags().String("cma-api-proxy-tls", "cma-api-proxy-tls", "Secret containing cma api proxy tls cert")
	rootCmd.Flags().String("cma-api-proxy", "", "DNS name for managed cluster api proxy")
	rootCmd.Flags().Bool("cma-insecure", true, "If we are not connecting to Cluster Manager API GRPC service over HTTPS")

	viper.BindPFlag("kubeconfig", rootCmd.Flags().Lookup("kubeconfig"))
	viper.BindPFlag("kubernetes-namespace", rootCmd.Flags().Lookup("kubernetes-namespace"))
	viper.BindPFlag("cma-endpoint", rootCmd.Flags().Lookup("cma-endpoint"))
	viper.BindPFlag("cma-api-proxy", rootCmd.Flags().Lookup("cma-api-proxy"))
	viper.BindPFlag("cma-api-proxy-tls", rootCmd.Flags().Lookup("cma-api-proxy-tls"))
	viper.BindPFlag("cma-insecure", rootCmd.Flags().Lookup("cma-insecure"))

	viper.AutomaticEnv()
	rootCmd.Flags().AddGoFlagSet(flag.CommandLine)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func operator() {
	var err error
	logger := util.GetModuleLogger("cmd.cma-operator", loggo.INFO)
	//viperInit()

	// get flags
	portNumber := viper.GetInt("port")
	kubeconfigLocation := viper.GetString("kubeconfig")

	// Debug for now
	logger.Infof("Parsed Variables: \n  Port: %d \n  Kubeconfig: %s", portNumber, kubeconfigLocation)

	k8sutil.KubeConfigLocation = kubeconfigLocation
	k8sutil.DefaultConfig, err = k8sutil.GenerateKubernetesConfig()

	if err != nil {
		logger.Infof("Was unable to generate a valid kubernetes default config, some functionality may be broken.  Error was %v", err)
	}

	// Install the CMA SDSCluster CRD
	k8sutil.CreateCRD(apiextensionsclient.NewForConfigOrDie(k8sutil.DefaultConfig), cma.GenerateSDSClusterCRD())
	// Install the CMA SDSPackageManager CRD
	k8sutil.CreateCRD(apiextensionsclient.NewForConfigOrDie(k8sutil.DefaultConfig), cma.GenerateSDSPackageManagerCRD())
	// Install the CMA SDSApplication CRD
	k8sutil.CreateCRD(apiextensionsclient.NewForConfigOrDie(k8sutil.DefaultConfig), cma.GenerateSDSApplicationCRD())
	// Install the CMA SDSAppBundle CRD
	k8sutil.CreateCRD(apiextensionsclient.NewForConfigOrDie(k8sutil.DefaultConfig), cma.GenerateSDSAppBundleCRD())

	var wg sync.WaitGroup
	stop := make(chan struct{})

	logger.Infof("Starting the SDSCluster Controller")
	sdsClusterController, err := sds_cluster.NewSDSClusterController(nil)
	if err != nil {
		logger.Criticalf("Could not create a sdsClusterController")
		os.Exit(-1)
	}
	sdsPackageManagerController, err := sds_package_manager.NewSDSPackageManagerController(nil)
	if err != nil {
		logger.Criticalf("Could not create a sdsPackageManagerController")
		os.Exit(-1)
	}
	sdsApplicationController, err := sdsapplication.NewSDSApplicationController(nil)
	if err != nil {
		logger.Criticalf("Could not create a sdsPackageManagerController")
		os.Exit(-1)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		sdsClusterController.Run(3, stop)
	}()

	// Start the SDSPackageManager Controller
	wg.Add(1)
	go func() {
		defer wg.Done()
		sdsPackageManagerController.Run(3, stop)
	}()
	// TODO: Start the SDSApplication Controller

	// Start the SDSPackageManager Controller
	wg.Add(1)
	go func() {
		defer wg.Done()
		sdsApplicationController.Run(3, stop)
	}()

	<-stop
	logger.Infof("Wating for controllers to shut down gracefully")
	wg.Wait()
}
