package sds_cluster

import (
	"fmt"
	"github.com/samsung-cnct/cma-operator/pkg/util/cmagrpc"
	"github.com/samsung-cnct/cma-operator/pkg/util/sds/callback"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	"k8s.io/apimachinery/pkg/fields"

	"github.com/golang/glog"
	"github.com/juju/loggo"
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
	WaitForClusterChangeMaxTries         = 3
	WaitForClusterChangeTimeInterval     = 5 * time.Second
	KubernetesNamespaceViperVariableName = "kubernetes-namespace"
	ClusterRequestIDAnnotation           = "requestID"
	ClusterCallbackURLAnnotation         = "callbackURL"
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
	} else {
		sdsCluster := obj.(*api.SDSCluster)
		clusterName := sdsCluster.GetName()
		fmt.Printf("SDSCluster %s does exist (name=%s)!\n", key, clusterName)

		switch sdsCluster.Status.Phase {
		case api.ClusterPhaseNone, api.ClusterPhasePending, api.ClusterPhaseWaitingForCluster, api.ClusterPhaseUpgrading:
			go c.waitForClusterReady(sdsCluster)
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
	retryCount := 0
	for retryCount < WaitForClusterChangeMaxTries {
		clusterInfo, err := cmagrpcClient.GetCluster(cmagrpc.GetClusterInput{Name: cluster.Name, Provider: cluster.Spec.Provider})
		if err == nil {
			logger.Errorf("Cluster status is %s", clusterInfo.Status)
			switch clusterInfo.Status {
			case "Created", "Succeeded", "Upgraded", "Ready":
				if c.handleClusterReady(cluster.Name, clusterInfo) {
					// We successfully handled it, apparently
					return
				}
			}
		} else {
			logger.Errorf("Error is %s for cluster %s", err, cluster.Name)
		}
		time.Sleep(WaitForClusterChangeTimeInterval)
		retryCount++
	}
	// We waited for the max number of retries, let's log it and bail
	logger.Errorf("waited too long for cluster -->%s<-- to work right", cluster.Name)
}

func (c *SDSClusterController) handleClusterReady(clusterName string, clusterInfo cmagrpc.GetClusterOutput) bool {
	freshCopy, err := c.client.CmaV1alpha1().SDSClusters(viper.GetString(KubernetesNamespaceViperVariableName)).Get(clusterName, v1.GetOptions{})
	if freshCopy.Annotations[ClusterRequestIDAnnotation] != "" {
		// We need to notify someone that the cluster is now ready (again)

		// Do Stuff here
		message := &sdscallback.ClusterMessage{
			State: sdscallback.ClusterMessageStateCompleted,
			Data: sdscallback.ClusterDataPayload{
				Kubeconfig:       clusterInfo.Kubeconfig,
				ClusterStatus:    clusterInfo.Status,
				CreationDateTime: string(freshCopy.ObjectMeta.CreationTimestamp.Unix()),
			},
		}
		sdscallback.DoCallback(freshCopy.Annotations[ClusterCallbackURLAnnotation], message)

		logger.Errorf("I was supposed to do something about cluster -->%s<--!", clusterName)

		freshCopy.Annotations[ClusterRequestIDAnnotation] = ""
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
