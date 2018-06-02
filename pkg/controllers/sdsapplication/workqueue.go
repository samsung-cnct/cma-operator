package sdsapplication

import (
	"fmt"
	"github.com/golang/glog"
	api "github.com/samsung-cnct/cma-operator/pkg/apis/cma/v1alpha1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"time"
)

func (c *SDSApplicationController) processNextItem() bool {
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
func (c *SDSApplicationController) processItem(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)

	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists {
		// Below we will warm up our cache with a SDSPackageManager, so that we will see a delete for one SDSPackageManager
		fmt.Printf("SDSApplication -->%s<-- does not exist anymore\n", key)
	} else {
		SDSApplication := obj.(*api.SDSApplication)
		clusterName := SDSApplication.GetName()
		fmt.Printf("SDSApplication -->%s<-- does exist (name=%s)!\n", key, clusterName)

		switch SDSApplication.Status.Phase {
		case api.ApplicationPhaseNone, api.ApplicationPhasePending:
			c.deployApplication(SDSApplication)
		case api.ApplicationPhaseInstalling:
			c.waitForApplication(SDSApplication)
		}
	}
	return nil
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *SDSApplicationController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.Infof("Error syncing SDSApplication %v: %v", key, err)

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

func (c *SDSApplicationController) Run(threadiness int, stopCh chan struct{}) {
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

func (c *SDSApplicationController) runWorker() {
	for c.processNextItem() {
	}
}
