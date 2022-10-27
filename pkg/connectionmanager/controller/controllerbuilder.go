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
	"context"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/handlers"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/google/uuid"
	api_v1 "k8s.io/api/core/v1"
	networking_v1 "k8s.io/api/networking/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

func NewPodController(client kubernetes.Interface) *Controller {
	var eventHandler handlers.Handler = new(handlers.PodHandler)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().Pods("").List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Pods("").Watch(context.TODO(), options)
			},
		},
		&api_v1.Pod{},
		0,
		cache.Indexers{},
	)
	return newResourceController(client, eventHandler, informer, constants.ResourceType_Pod)
}

func NewServiceController(client kubernetes.Interface) *Controller {
	var eventHandler handlers.Handler = new(handlers.ServiceHandler)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().Services("").List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Services("").Watch(context.TODO(), options)
			},
		},
		&api_v1.Service{},
		0,
		cache.Indexers{},
	)
	return newResourceController(client, eventHandler, informer, constants.ResourceType_Service)
}

func NewEndpointsController(client kubernetes.Interface) *Controller {
	var eventHandler handlers.Handler = new(handlers.EndpointsHandler)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().Endpoints("").List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Endpoints("").Watch(context.TODO(), options)
			},
		},
		&api_v1.Endpoints{},
		0,
		cache.Indexers{},
	)
	return newResourceController(client, eventHandler, informer, constants.ResourceType_Endpoints)
}

func NewNodeController(client kubernetes.Interface) *Controller {
	var eventHandler handlers.Handler = new(handlers.NodeHandler)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().Nodes().List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().Nodes().Watch(context.TODO(), options)
			},
		},
		&api_v1.Node{},
		0,
		cache.Indexers{},
	)
	return newResourceController(client, eventHandler, informer, constants.ResourceType_Node)
}

func NewSystemConfigMapController(client kubernetes.Interface) *Controller {
	var eventHandler handlers.Handler = new(handlers.ConfigMapHandler)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.CoreV1().ConfigMaps("kube-system").List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.CoreV1().ConfigMaps("kube-system").Watch(context.TODO(), options)
			},
		},
		&api_v1.ConfigMap{},
		0,
		cache.Indexers{},
	)
	return newResourceController(client, eventHandler, informer, constants.ResourceType_SystemConfigMap)
}

func NewIngressController(client kubernetes.Interface) *Controller {
	var eventHandler handlers.Handler = new(handlers.IngressHandler)
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				return client.NetworkingV1().Ingresses("").List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				return client.NetworkingV1().Ingresses("").Watch(context.TODO(), options)
			},
		},
		&networking_v1.Ingress{},
		0,
		cache.Indexers{},
	)
	return newResourceController(client, eventHandler, informer, constants.ResourceType_Ingress)
}

func newResourceController(client kubernetes.Interface, eventHandler handlers.Handler, informer cache.SharedIndexInformer, resourceType string) *Controller {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	var eventItem objects.EventItem
	var err error
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			eventItem.Key, err = cache.MetaNamespaceKeyFunc(obj)
			eventItem.EventType = constants.EventType_Set
			eventItem.Id = uuid.New().String()
			klog.Infof("Processing add to %v: %s. Tracking Id: %s", resourceType, eventItem.Key, eventItem.Id)
			if err == nil {
				queue.Add(eventItem)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			eventItem.Key, err = cache.MetaNamespaceKeyFunc(old)
			eventItem.EventType = constants.EventType_Set
			eventItem.Id = uuid.New().String()
			klog.Infof("Processing update to %v: %s. Tracking Id: %s", resourceType, eventItem.Key, eventItem.Id)
			if err == nil {
				queue.Add(eventItem)
			}
		},
		DeleteFunc: func(obj interface{}) {
			eventItem.Key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			eventItem.EventType = constants.EventType_Delete
			eventItem.Id = uuid.New().String()
			klog.Infof("Processing delete to %v: %s. Tracking Id: %s", resourceType, eventItem.Key, eventItem.Id)
			if err == nil {
				queue.Add(eventItem)
			}
		},
	})

	return &Controller{
		resourceType: resourceType,
		clientset:    client,
		informer:     informer,
		queue:        queue,
		eventHandler: eventHandler,
	}
}
