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
	"encoding/json"
	"reflect"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/google/uuid"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

func calculateNextEndpointsResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.EndpointsResourceVersion += 1
	return instance.EndpointsResourceVersion
}

func SetEndpoints(name string, ipWithPorts []string, trackingId string) {
	endpointsMap := Instance().EndpointsMap
	endpoints, exists := endpointsMap[name]
	changed := false
	if exists {
		if !reflect.DeepEqual(endpoints.IPWithPorts, ipWithPorts) {
			changed = true
		} else {
			klog.Infof("Handling endpoints completed. Endpoints %s is unchanged. Tracking Id: %s", name, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		resourceVersion := calculateNextEndpointsResourceVersion()
		newEndpoints := &objects.EndpointsObject{
			Name:            name,
			IPWithPorts:     ipWithPorts,
			ResourceVersion: resourceVersion,
		}
		newEndpointsEvent := &objects.EndpointsEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Set,
			EndpointsObject: *newEndpoints,
		}
		endpointsMap[name] = newEndpoints
		Instance().EndpointsEventMap[resourceVersion] = newEndpointsEvent
		EnqueueEndpoints(*newEndpointsEvent)

		endpointsStr, _ := json.Marshal(endpointsMap[name])
		klog.Infof("Handling endpoints completed. Endpoints set as %s. Tracking Id: %s", endpointsStr, trackingId)
	}
}

func DeleteEndpoints(name string, trackingId string) {
	endpointsMap := Instance().EndpointsMap
	endpoints, exists := endpointsMap[name]
	if exists {
		resourceVersion := calculateNextEndpointsResourceVersion()
		newEndpointsEvent := &objects.EndpointsEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Delete,
			EndpointsObject: *endpoints,
		}
		Instance().EndpointsEventMap[resourceVersion] = newEndpointsEvent
		EnqueueEndpoints(*newEndpointsEvent)
		delete(endpointsMap, name)
		klog.Infof("Handling endpoints completed. Endpoints %s is deleted. Tracking Id: %s", name, trackingId)
	}
}

func ListEndpoints(minResourceVersion int) []objects.EndpointsEventObject {
	maxResourceVersion := Instance().EndpointsResourceVersion
	endpointsEventMap := Instance().EndpointsEventMap

	var endpointsEvents []objects.EndpointsEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		endpointsEvents = append(endpointsEvents, *endpointsEventMap[i])
	}
	return endpointsEvents
}

func EnqueueEndpoints(endpointsEventObject objects.EndpointsEventObject) {
	for _, queue := range Instance().EndpointsQueueMap {
		queue.Add(endpointsEventObject)
	}
}

func AddEndpointsQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().EndpointsQueueMap[key] = queue
}

func RemoveEndpointsQueue(key uuid.UUID) {
	delete(Instance().EndpointsQueueMap, key)
}
