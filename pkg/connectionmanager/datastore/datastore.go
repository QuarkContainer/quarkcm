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

package datastore

import (
	"sync"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/google/uuid"
	"k8s.io/client-go/util/workqueue"
)

type DataStore struct {
	NodeResourceVersion int
	NodeMap             map[string]*objects.NodeObject                // map[node name] => node object
	NodeEventMap        map[int]*objects.NodeEventObject              // map[resource version] => node event object
	NodeQueueMap        map[uuid.UUID]workqueue.RateLimitingInterface // map[guid] => node queue
	NodeIPMap           map[string]bool                               // map[node ip] => not using, this map is used as hashset

	PodResourceVersion int
	PodMap             map[string]*objects.PodObject                 // map[key] => pod object
	PodEventMap        map[int]*objects.PodEventObject               // map[resource version] => pod event object
	PodQueueMap        map[uuid.UUID]workqueue.RateLimitingInterface // map[guid] => pod queue

	ServiceResourceVersion int
	ServiceMap             map[string]*objects.ServiceObject             // map[key] => service object
	ServiceEventMap        map[int]*objects.ServiceEventObject           // map[resource version] => service event object
	ServiceQueueMap        map[uuid.UUID]workqueue.RateLimitingInterface // map[guid] => service queue

	EndpointsResourceVersion int
	EndpointsMap             map[string]*objects.EndpointsObject           // map[key] => endpoints object
	EndpointsEventMap        map[int]*objects.EndpointsEventObject         // map[resource version] => endpoints event object
	EndpointsQueueMap        map[uuid.UUID]workqueue.RateLimitingInterface // map[guid] => endpoints queue
}

var lock = &sync.Mutex{}
var dataStore *DataStore

func Instance() *DataStore {
	if dataStore == nil {
		lock.Lock()
		defer lock.Unlock()
		if dataStore == nil {
			dataStore = &DataStore{
				NodeResourceVersion: 0,
				NodeMap:             map[string]*objects.NodeObject{},
				NodeEventMap:        map[int]*objects.NodeEventObject{},
				NodeQueueMap:        map[uuid.UUID]workqueue.RateLimitingInterface{},
				NodeIPMap:           map[string]bool{},

				PodResourceVersion: 0,
				PodMap:             map[string]*objects.PodObject{},
				PodEventMap:        map[int]*objects.PodEventObject{},
				PodQueueMap:        map[uuid.UUID]workqueue.RateLimitingInterface{},

				ServiceResourceVersion: 0,
				ServiceMap:             map[string]*objects.ServiceObject{},
				ServiceEventMap:        map[int]*objects.ServiceEventObject{},
				ServiceQueueMap:        map[uuid.UUID]workqueue.RateLimitingInterface{},

				EndpointsResourceVersion: 0,
				EndpointsMap:             map[string]*objects.EndpointsObject{},
				EndpointsEventMap:        map[int]*objects.EndpointsEventObject{},
				EndpointsQueueMap:        map[uuid.UUID]workqueue.RateLimitingInterface{},
			}
		}
	}
	return dataStore
}
