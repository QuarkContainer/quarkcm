/*
Copyright 2017 The Kubernetes Authors.

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

package rdmaingresscontroller

import (
	"fmt"
	"os"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/handlers"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	clientset "github.com/CentaurusInfra/quarkcm/pkg/generated/clientset/versioned"
	rdmaingressscheme "github.com/CentaurusInfra/quarkcm/pkg/generated/clientset/versioned/scheme"
	informersFactory "github.com/CentaurusInfra/quarkcm/pkg/generated/informers/externalversions"
	informers "github.com/CentaurusInfra/quarkcm/pkg/generated/informers/externalversions/rdmaingresscontroller/v1alpha1"
	listers "github.com/CentaurusInfra/quarkcm/pkg/generated/listers/rdmaingresscontroller/v1alpha1"
	"github.com/google/uuid"
)

// RdmaController is the controller implementation for RdmaIngress resources
type RdmaController struct {
	// rdmaingressclientset is a clientset for our own API group
	rdmaingressclientset clientset.Interface

	rdmaIngressesLister listers.RdmaIngressLister
	rdmaIngressesSynced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface

	eventHandler handlers.Handler
}

// NewController returns a new rdmaingress controller
func NewRdmaController(
	rdmaingressclientset clientset.Interface,
	rdmaIngressInformer informers.RdmaIngressInformer) *RdmaController {

	// Create event broadcaster
	// Add rdmaingress-controller types to the default Kubernetes Scheme so Events can be
	// logged for rdmaingress-controller types.
	utilruntime.Must(rdmaingressscheme.AddToScheme(scheme.Scheme))

	controller := &RdmaController{
		rdmaingressclientset: rdmaingressclientset,
		rdmaIngressesLister:  rdmaIngressInformer.Lister(),
		rdmaIngressesSynced:  rdmaIngressInformer.Informer().HasSynced,
		workqueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RdmaIngresses"),
		eventHandler:         new(handlers.RdmaIngressHandler),
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when RdmaIngress resources change
	rdmaIngressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueueRdmaIngress(obj, constants.EventType_Set)
		},
		UpdateFunc: func(old, obj interface{}) {
			controller.enqueueRdmaIngress(obj, constants.EventType_Set)
		},
		DeleteFunc: func(obj interface{}) {
			controller.enqueueRdmaIngress(obj, constants.EventType_Delete)
		},
	})

	return controller
}

func Start() {
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = buildOutOfClusterConfig()
		if err != nil {
			klog.Fatalf("Can not get kubernetes config: %v", err)
		}
	}

	rdmaIngressClient, err := clientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building rdmaIngress clientset: %s", err.Error())
	}

	rdmaIngressInformerFactory := informersFactory.NewSharedInformerFactory(rdmaIngressClient, time.Second*30)

	controller := NewRdmaController(rdmaIngressClient,
		rdmaIngressInformerFactory.Rdmaingresscontroller().V1alpha1().RdmaIngresses())

	stopCh := make(chan struct{})
	defer close(stopCh)
	rdmaIngressInformerFactory.Start(stopCh)

	if err = controller.Run(1, stopCh); err != nil {
		klog.Fatalf("Error running RdmaIngress controller: %s", err.Error())
	}
}

func buildOutOfClusterConfig() (*rest.Config, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *RdmaController) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting RdmaIngress controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for RdmaIngress informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.rdmaIngressesSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting RdmaIngress workers")
	// Launch two workers to process RdmaIngress resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started RdmaIngress workers")
	<-stopCh
	klog.Info("Shutting down RdmaIngress workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *RdmaController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue
func (c *RdmaController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	c.eventHandler.Handle(obj.(objects.EventItem))
	c.workqueue.Forget(obj)

	return true
}

func (c *RdmaController) enqueueRdmaIngress(obj interface{}, eventType string) {
	var eventItem objects.EventItem
	var err error
	eventItem.Key, err = cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	eventItem.EventType = eventType
	eventItem.Id = uuid.New().String()
	eventItem.Obj = obj
	c.workqueue.Add(eventItem)
}
