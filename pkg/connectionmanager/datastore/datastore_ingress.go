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

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/google/uuid"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

func calculateNextIngressResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.IngressResourceVersion += 1
	return instance.IngressResourceVersion
}

func SetIngress(name string, service string, portNumber uint16, trackingId string) {
	ingressMap := Instance().IngressMap
	ingress, exists := ingressMap[name]
	changed := false
	if exists {
		if service != ingress.Service || portNumber != ingress.PortNumber {
			changed = true
		} else {
			klog.Infof("Handling ingress completed. Ingress %s is unchanged. Tracking Id: %s", name, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		resourceVersion := calculateNextIngressResourceVersion()
		newIngress := &objects.IngressObject{
			Name:            name,
			Service:         service,
			PortNumber:      portNumber,
			ResourceVersion: resourceVersion,
		}
		newIngressEvent := &objects.IngressEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Set,
			IngressObject:   *newIngress,
		}
		ingressMap[name] = newIngress
		Instance().IngressEventMap[resourceVersion] = newIngressEvent
		EnqueueIngress(*newIngressEvent)

		ingressStr, _ := json.Marshal(ingressMap[name])
		klog.Infof("Handling ingress completed. Ingress set as %s. Tracking Id: %s", ingressStr, trackingId)
	}
}

func DeleteIngress(name string, trackingId string) {
	ingressMap := Instance().IngressMap
	ingress, exists := ingressMap[name]
	if exists {
		resourceVersion := calculateNextIngressResourceVersion()
		newIngressEvent := &objects.IngressEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Delete,
			IngressObject:   *ingress,
		}
		Instance().IngressEventMap[resourceVersion] = newIngressEvent
		EnqueueIngress(*newIngressEvent)
		delete(ingressMap, name)
		klog.Infof("Handling ingress completed. Ingress %s is deleted. Tracking Id: %s", name, trackingId)
	}
}

func ListIngress(minResourceVersion int) []objects.IngressEventObject {
	maxResourceVersion := Instance().IngressResourceVersion
	ingressEventMap := Instance().IngressEventMap

	var ingressEvents []objects.IngressEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		ingressEvents = append(ingressEvents, *ingressEventMap[i])
	}
	return ingressEvents
}

func EnqueueIngress(ingressEventObject objects.IngressEventObject) {
	for _, queue := range Instance().IngressQueueMap {
		queue.Add(ingressEventObject)
	}
}

func AddIngressQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().IngressQueueMap[key] = queue
}

func RemoveIngressQueue(key uuid.UUID) {
	delete(Instance().IngressQueueMap, key)
}
