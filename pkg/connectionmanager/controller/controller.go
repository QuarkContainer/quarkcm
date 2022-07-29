/*
Copyright 2022 quarkcm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/handlers"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/utils"
	"k8s.io/klog"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const maxRetries = 5

type Controller struct {
	resourceType string
	clientset    kubernetes.Interface
	queue        workqueue.RateLimitingInterface
	informer     cache.SharedIndexInformer
	eventHandler handlers.Handler
}

func Start() {
	var kubeClient kubernetes.Interface

	if _, err := rest.InClusterConfig(); err != nil {
		kubeClient = utils.GetClientOutOfCluster()
	} else {
		kubeClient = utils.GetClient()
	}

	nodeController := NewNodeController(kubeClient)
	nodeStopCh := make(chan struct{})
	defer close(nodeStopCh)

	nodeController.Start(nodeStopCh) // Sync start node controller first, to generating node ip set for filtering pod ip
	go nodeController.Wait(nodeStopCh)

	podController := NewPodController(kubeClient)
	podStopCh := make(chan struct{})
	defer close(podStopCh)

	go podController.Run(podStopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}

func (c *Controller) Run(stopCh <-chan struct{}) {
	c.Start(stopCh)
	c.Wait(stopCh)
}

func (c *Controller) Start(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	klog.Infof("Starting quarkcm %s controller", c.resourceType)
	go c.informer.Run(stopCh)

	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		utilruntime.HandleError(fmt.Errorf("%s controller timed out waiting for caches to sync", c.resourceType))
		return
	}
	klog.Infof("quarkcm %s controller synced and ready", c.resourceType)
}

func (c *Controller) Wait(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	wait.Until(c.runWorker, time.Second, stopCh)
}

// HasSynced is required for the cache.Controller interface.
func (c *Controller) HasSynced() bool {
	return c.informer.HasSynced()
}

// LastSyncResourceVersion is required for the cache.Controller interface.
func (c *Controller) LastSyncResourceVersion() string {
	return c.informer.LastSyncResourceVersion()
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
		// continue looping
	}
}

func (c *Controller) processNextItem() bool {
	queueItem, quit := c.queue.Get()
	eventItem := queueItem.(objects.EventItem)

	if quit {
		return false
	}
	defer c.queue.Done(queueItem)
	err := c.processItem(eventItem)
	if err == nil {
		// No error, reset the ratelimit counters
		c.queue.Forget(queueItem)
	} else if c.queue.NumRequeues(queueItem) < maxRetries {
		klog.Errorf("error processing %s (will retry): %v. Tracking Id: %s", eventItem.Key, err, eventItem.Id)
		c.queue.AddRateLimited(queueItem)
	} else {
		// err != nil and too many retries
		klog.Errorf("error processing %s (giving up): %v. Tracking Id: %s", eventItem.Key, err, eventItem.Id)
		c.queue.Forget(queueItem)
		utilruntime.HandleError(err)
	}

	return true
}

func (c *Controller) processItem(eventItem objects.EventItem) error {
	obj, _, err := c.informer.GetIndexer().GetByKey(eventItem.Key)
	if err != nil {
		return fmt.Errorf("error fetching object with key %s from store: %v. Tracking Id: %s", eventItem.Key, err, eventItem.Id)
	}
	eventItem.Obj = obj

	// namespace retrived from event key incase namespace value is empty
	if eventItem.Namespace == "" && strings.Contains(eventItem.Key, "/") {
		substring := strings.Split(eventItem.Key, "/")
		eventItem.Namespace = substring[0]
		eventItem.Key = substring[1]
	}

	c.eventHandler.Handle(eventItem)
	return nil
}
