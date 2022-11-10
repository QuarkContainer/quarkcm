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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	clientset "github.com/CentaurusInfra/quarkcm/pkg/generated/clientset/versioned"
	rdmaingressscheme "github.com/CentaurusInfra/quarkcm/pkg/generated/clientset/versioned/scheme"
	informersFactory "github.com/CentaurusInfra/quarkcm/pkg/generated/informers/externalversions"
	informers "github.com/CentaurusInfra/quarkcm/pkg/generated/informers/externalversions/rdmaingresscontroller/v1alpha1"
	listers "github.com/CentaurusInfra/quarkcm/pkg/generated/listers/rdmaingresscontroller/v1alpha1"
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
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when RdmaIngress resources change
	rdmaIngressInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.handleObject,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueRdmaIngress(new)
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

	if err = controller.Run(2, stopCh); err != nil {
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
	// <-stopCh
	// klog.Info("Shutting down RdmaIngress workers")

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

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// enqueueRdmaIngress takes a RdmaIngress resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than RdmaIngress.
func (c *RdmaController) enqueueRdmaIngress(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the RdmaIngress resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that RdmaIngress resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *RdmaController) handleObject(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object, invalid type"))
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
			return
		}
		klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
	}
	klog.Infof("Processing object: %s", object.GetName())
}
