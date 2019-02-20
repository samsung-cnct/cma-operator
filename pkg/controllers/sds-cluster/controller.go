package sds_cluster

import (
	"encoding/json"
	"fmt"
	"github.com/docker/distribution/uuid"
	"github.com/samsung-cnct/cma-operator/pkg/util/cmagrpc"
	"github.com/samsung-cnct/cma-operator/pkg/util/sds/callback"
	"github.com/samsung-cnct/cma-operator/pkg/util/sds/token"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/golang/glog"
	"github.com/juju/loggo"
	pb "github.com/samsung-cnct/cluster-manager-api/pkg/generated/api"
	api "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"github.com/samsung-cnct/cma-operator/pkg/generated/cma/client/clientset/versioned"
	"github.com/samsung-cnct/cma-operator/pkg/util"
	"github.com/samsung-cnct/cma-operator/pkg/util/k8sutil"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	WaitForClusterChangeMaxTries         = 100
	WaitForClusterChangeTimeInterval     = 30 * time.Second
	KubernetesNamespaceViperVariableName = "kubernetes-namespace"
	ClusterRequestIDAnnotation           = "requestID"
	ClusterCallbackURLAnnotation         = "callbackURL"
	LoggingPackageManagerName            = "pm-logger"
	LoggingApplicationName               = "logging-client"
	LoggingNamespace                     = "logging"
	MetricServerApplicationName          = "metrics-server"
	KubeSystemNamespace                  = "kube-system"
	KubeSystemPackageManagerName         = "pm-kube-system"
	StateMetricsApplicationName          = "kube-state-metrics"
	NodeLabelBot5000ApplicationName      = "nodelabelbot5000"
	ApiEndpointIngressName               = "k8s-api-ingress-for"
	ApiEndpointBackendServiceName        = "k8s-api-service-for"
	MonitoringProxyApplicationName       = "nginx-k8smon"
)

var (
	logger loggo.Logger
)

type SDSClusterController struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller

	client        *versioned.Clientset
	cmaGRPCClient cmagrpc.ClientInterface
}

func NewSDSClusterController(config *rest.Config) (*SDSClusterController, error) {
	cmaGRPCClient, err := cmagrpc.CreateNewDefaultClient()
	if err != nil {
		return nil, err
	}
	if config == nil {
		config = k8sutil.DefaultConfig
	}
	client := versioned.NewForConfigOrDie(config)

	// create sdscluster list/watcher
	sdsClusterListWatcher := cache.NewListWatchFromClient(
		client.CmaV1alpha1().RESTClient(),
		api.SDSClusterResourcePlural,
		viper.GetString(KubernetesNamespaceViperVariableName),
		fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Bind the workqueue to a cache with the help of an informer. This way we make sure that
	// whenever the cache is updated, the SDSCluster key is added to the workqueue.
	// Note that when we finally process the item from the workqueue, we might see a newer version
	// of the SDSCluster than the version which was responsible for triggering the update.
	indexer, informer := cache.NewIndexerInformer(sdsClusterListWatcher, &api.SDSCluster{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	output := &SDSClusterController{
		informer:      informer,
		indexer:       indexer,
		queue:         queue,
		client:        client,
		cmaGRPCClient: cmaGRPCClient,
	}
	output.SetLogger()
	return output, nil
}

func (c *SDSClusterController) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two SDSClusters with the same key are never processed in
	// parallel.
	defer c.queue.Done(key)

	// Invoke the method containing the business logic
	err := c.processItem(key.(string))
	// Handle the error if something went wrong during the execution of the business logic
	c.handleErr(err, key)
	return true
}

// processItem is the business logic of the controller.
func (c *SDSClusterController) processItem(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a SDSCluster, so that we will see a delete for one SDSCluster
		fmt.Printf("SDSCluster %s does not exist anymore\n", key)
		c.queue.Forget(key)
	} else {
		sdsCluster := obj.(*api.SDSCluster)
		clusterName := sdsCluster.GetName()
		fmt.Printf("Examining Cluster -->%s<--", clusterName)

		switch sdsCluster.Status.Phase {
		case api.ClusterPhaseNone, api.ClusterPhasePending, api.ClusterPhaseWaitingForCluster:
			go c.waitForClusterReady(sdsCluster)
			break
		case api.ClusterPhaseDeleting:
			go c.handleDeletedCluster(sdsCluster)
			break
		case api.ClusterPhaseFailed:
			go c.handleFailedCluster(sdsCluster)
			break
		case api.ClusterPhaseUpgrading:
			go c.handleUpgradedCluster(sdsCluster)
			break
		}
	}
	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *SDSClusterController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.Infof("Error syncing krakenCluster %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.Infof("Dropping krakenCluster %q out of the queue: %v", key, err)
}

func (c *SDSClusterController) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	glog.Info("Starting SDSCluster controller")

	go c.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping SDSCluster controller")
}

func (c *SDSClusterController) runWorker() {
	for c.processNextItem() {
	}
}

func (c *SDSClusterController) SetLogger() {
	logger = util.GetModuleLogger("pkg.controllers.sds_cluster", loggo.INFO)
}

func (c *SDSClusterController) waitForClusterReady(cluster *api.SDSCluster) {
	cmagrpcClient, err := cmagrpc.CreateNewDefaultClient()
	if err != nil {
		logger.Errorf("could not create cma grpc client while waiting for cluster %s to come up", cluster.Name)
	}
	if cluster.Annotations[ClusterCallbackURLAnnotation] != "" {
		// We need to notify someone that the cluster is now in progress (again)
		message := &sdscallback.ClusterMessage{
			State:        sdscallback.ClusterMessageStateInProgress,
			StateText:    pb.ClusterStatus_PROVISIONING.String(),
			ProgressRate: 0,
		}
		sdscallback.DoCallback(cluster.Annotations[ClusterCallbackURLAnnotation], message)
	}
	retryCount := 0
	for retryCount < WaitForClusterChangeMaxTries {
		clusterInfo, err := cmagrpcClient.GetCluster(cmagrpc.GetClusterInput{Name: cluster.Name, Provider: cluster.Spec.Provider})
		if err == nil {
			logger.Errorf("Cluster status is %s", clusterInfo.Status)
			switch clusterInfo.Status {
			case "Created", "Succeeded", "Upgraded", "Ready", pb.ClusterStatus_RUNNING.String():
				if c.handleClusterReady(cluster.Name, clusterInfo) {
					// We successfully handled it, apparently
					return
				}
			case pb.ClusterStatus_ERROR.String():
				// set cluster status to failed
				cluster.Status.Phase = api.ClusterPhaseFailed
				_, err = c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Update(cluster)
				if err != nil {
					logger.Errorf("error updating cluster -->%s<-- status to failed, error message: %s", cluster.Name, err)
				}
				return
			}
		} else {
			logger.Errorf("Error is %s for cluster %s", err, cluster.Name)
		}
		time.Sleep(WaitForClusterChangeTimeInterval)
		retryCount++
	}
	// We waited for the max number of retries, let's log it and bail
	logger.Errorf("waited too long for cluster -->%s<-- to work right", cluster.Name)
	// set cluster status to failed
	cluster.Status.Phase = api.ClusterPhaseFailed
	_, err = c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Update(cluster)
	if err != nil {
		logger.Errorf("error updating cluster -->%s<-- status to failed, error message: %s", cluster.Name, err)
	}

}

func (c *SDSClusterController) handleClusterReady(clusterName string, clusterInfo cmagrpc.GetClusterOutput) bool {
	freshCopy, err := c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Get(clusterName, v1.GetOptions{})
	if freshCopy.Annotations[ClusterCallbackURLAnnotation] != "" {

		// check all SDSAppBundles
		bundleErr := c.checkBundles(freshCopy)
		if bundleErr != nil {
			logger.Errorf("something bad happened when checking SDSAppBundles for cluster -->%s<-- error: %s", clusterName, err)
		}
		// check if logging package manager exists
		clusterLoggingPackageManagerName := LoggingPackageManagerName + "-" + clusterName
		_, err := c.client.CmaV1alpha1().SDSPackageManagers(viper.GetString(KubernetesNamespaceViperVariableName)).Get(clusterLoggingPackageManagerName, v1.GetOptions{})
		if err != nil {
			logger.Errorf("the logging package manager for cluster -->%s<-- does not exist! creating it", clusterName)

			// create sdsPackageManager for logging
			loggingPackageManager := &api.SDSPackageManager{
				Spec: api.SDSPackageManagerSpec{
					Name:      LoggingPackageManagerName,
					Namespace: LoggingNamespace,
					Version:   "v2.11.0",
					Image:     "gcr.io/kubernetes-helm/tiller",
					ServiceAccount: api.ServiceAccount{
						Name:      LoggingPackageManagerName,
						Namespace: LoggingNamespace,
					},
					Permissions: api.PackageManagerPermissions{
						ClusterWide: true,
						Namespaces: []string{
							LoggingNamespace,
						},
					},
					Cluster: api.SDSClusterRef{
						Name: clusterName,
					},
				},
			}

			loggingPackageManager.Name = loggingPackageManager.Spec.Name + "-" + clusterName
			loggingPackageManager.Namespace = viper.GetString(KubernetesNamespaceViperVariableName)

			newLoggerPackageManager, err := c.client.CmaV1alpha1().SDSPackageManagers(viper.GetString(KubernetesNamespaceViperVariableName)).Create(loggingPackageManager)
			if err != nil {
				logger.Errorf("something bad happened when creating logging package manager for cluster -->%s<-- error: %s", clusterName, err)
			}
			logger.Infof("create logging package manager -->%s<-- for cluster -->%s<--", newLoggerPackageManager.Name, clusterName)
		}

		// check if logging application exists
		clusterLoggingApplicationName := LoggingApplicationName + "-" + LoggingPackageManagerName + "-" + clusterName
		_, err = c.client.CmaV1alpha1().SDSApplications(viper.GetString(KubernetesNamespaceViperVariableName)).Get(clusterLoggingApplicationName, v1.GetOptions{})
		if err != nil {
			logger.Errorf("the logging application for cluster -->%s<-- does not exist, we should create it,", clusterName)

			// create sdsApplication for logging
			// TODO: get the ElasticSearchHost
			// TODO: get the elasticSearch Password
			uuidForLogging := uuid.Generate()

			loggerApplication := &api.SDSApplication{
				Spec: api.SDSApplicationSpec{
					PackageManager: api.SDSPackageManagerRef{
						Name: LoggingPackageManagerName,
					},
					Namespace: LoggingNamespace,
					Name:      LoggingApplicationName,
					Chart: api.Chart{
						Name: LoggingApplicationName,
						Repository: api.ChartRepository{
							Name: LoggingApplicationName,
							URL:  "https://charts.cnct.io",
						},
					},
					Values: "fluent-bit:\n name: fluent-bit\n cluster_uuid: " + uuidForLogging.String() + "\n elasticSearchHost: es.aws.uswest1.hybridstack.cnct.io\n elasticSearchPassword: changeme",
					Cluster: api.SDSClusterRef{
						Name: clusterName,
					},
				},
			}
			loggerApplication.Name = clusterLoggingApplicationName
			loggerApplication.Namespace = viper.GetString(KubernetesNamespaceViperVariableName)
			newLoggerApplication, err := c.client.CmaV1alpha1().SDSApplications(viper.GetString(KubernetesNamespaceViperVariableName)).Create(loggerApplication)
			if err != nil {
				logger.Errorf("something bad happened when creating the logging application for cluster -->%s<-- error: %s", clusterName, err)
			}
			logger.Infof("create logging application -->%s<-- for cluster -->%s<--", newLoggerApplication.Name, clusterName)
		}
		// End of logging

		// parse the kubeconfig string for api endpoint
		// ingress and service for managed cluster api server
		clusterApiEndpointServiceName := ApiEndpointBackendServiceName + "-" + clusterName
		apiEndpoint, err := k8sutil.GetClusterEndpoint(clusterInfo.Kubeconfig)
		if err != nil {
			logger.Errorf("something bad happened when getting api endpoint for cluster -->%s<-- error: %s", clusterName, err)
		}

		_, err = k8sutil.CreateExternalService(
			k8sutil.GenerateExternalService(clusterApiEndpointServiceName, apiEndpoint),
			freshCopy, nil)
		if err != nil {
			logger.Errorf("something bad happened when creating the service for cluster -->%s<-- error: %s", clusterName, err)
		}

		clusterApiEndpointIngressName := ApiEndpointIngressName + "-" + clusterName
		_, err = k8sutil.CreateIngress(
			k8sutil.GenerateIngress(clusterApiEndpointIngressName, clusterName, clusterApiEndpointServiceName),
			freshCopy, nil)
		if err != nil {
			logger.Errorf("something bad happened when creating the ingress for cluster -->%s<-- error: %s", clusterName, err)
		}

		// bearer token service account
		sdsTokenClient, err := sdstoken.NewSDSTokenClient(nil)
		if err != nil {
			logger.Errorf("unable to create new sds token client", err)
		}
		token, err := sdsTokenClient.CreateSDSToken(freshCopy, viper.GetString(KubernetesNamespaceViperVariableName))
		if err != nil {
			logger.Errorf("unable to create sds token, error message: %s", err)
		}
		clusterInfo.Bearertoken = string(token)
		// End bearer token service account

		// We need to notify someone that the cluster is now ready (again)
		dataPayload, _ := json.Marshal(sdscallback.ClusterDataPayload{
			Bearertoken:      clusterInfo.Bearertoken,
			Kubeconfig:       clusterInfo.Kubeconfig,
			ClusterStatus:    clusterInfo.Status,
			CreationDateTime: string(freshCopy.ObjectMeta.CreationTimestamp.Unix()),
		})

		// Do Stuff here
		message := &sdscallback.ClusterMessage{
			State:        sdscallback.ClusterMessageStateCompleted,
			StateText:    clusterInfo.Status,
			ProgressRate: 100,
			Data:         string(dataPayload),
		}
		sdscallback.DoCallback(freshCopy.Annotations[ClusterCallbackURLAnnotation], message)

		logger.Errorf("I was supposed to do something about cluster -->%s<--!", clusterName)
	} else {
		logger.Errorf("No annotation on cluster -->%s<--", clusterName)
	}
	if err == nil {
		freshCopy.Status.Phase = api.ClusterPhaseReady
		_, err = c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Update(freshCopy)
		if err == nil {
			return true
		}
		logger.Errorf("I was supposed to do something about cluster -->%s<--!", err)
	}
	// Something happened, so let's sleep and try again
	return false
}

func (c *SDSClusterController) handleDeletedCluster(cluster *api.SDSCluster) {
	cmagrpcClient, err := cmagrpc.CreateNewDefaultClient()
	if err != nil {
		logger.Errorf("could not create cma grpc client while waiting for cluster %s to delete", cluster.Name)
	}
	retryCount := 0
	for retryCount < WaitForClusterChangeMaxTries {
		_, err := cmagrpcClient.GetCluster(cmagrpc.GetClusterInput{Name: cluster.Name, Provider: cluster.Spec.Provider})
		if err != nil {
			logger.Errorf("Cluster %s is presumed to be deleted", cluster.Name)
			message := &sdscallback.ClusterMessage{
				State:        sdscallback.ClusterMessageStateCompleted,
				StateText:    "Deleted",
				ProgressRate: 100,
			}
			sdscallback.DoCallback(cluster.Annotations[ClusterCallbackURLAnnotation], message)
			c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Delete(cluster.Name, &v1.DeleteOptions{})
			return
		}
		time.Sleep(WaitForClusterChangeTimeInterval)
		retryCount++
	}
	// We waited for the max number of retries, let's log it and bail
	logger.Errorf("waited too long for cluster -->%s<-- to work right", cluster.Name)

}

func (c *SDSClusterController) handleFailedCluster(cluster *api.SDSCluster) {
	// We need to notify someone that the cluster has failed
	dataPayload, _ := json.Marshal(sdscallback.ClusterDataPayload{
		ClusterStatus:    api.ClusterPhaseFailed,
	})

	message := &sdscallback.ClusterMessage{
		State:        sdscallback.ClusterMessageStateFailed,
		StateText:    api.ClusterPhaseFailed,
		ProgressRate: 0,
		Data:         string(dataPayload),
	}
	sdscallback.DoCallback(cluster.Annotations[ClusterCallbackURLAnnotation], message)

	// setting cluster status to error
	cluster.Status.Phase = "Error"
	_, err := c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Update(cluster)
	if err != nil {
		logger.Errorf("error updating cluster -->%s<-- status to 'Error', error message: %s", cluster.Name, err)
	}
}

func (c *SDSClusterController) handleUpgradedCluster(cluster *api.SDSCluster) {

}
