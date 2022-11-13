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

func calculateNextRdmaIngressResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.RdmaIngressResourceVersion += 1
	return instance.RdmaIngressResourceVersion
}

func SetRdmaIngress(portNumber uint16, service string, targetPortNumber uint16, trackingId string) {
	rdmaIngressMap := Instance().RdmaIngressMap
	rdmaIngress, exists := rdmaIngressMap[portNumber]
	changed := false
	if exists {
		if service != rdmaIngress.Service || portNumber != rdmaIngress.PortNumber {
			changed = true
		} else {
			klog.Infof("Handling rdmaingress completed. RdmaIngress with port %s is unchanged. Tracking Id: %s", portNumber, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		resourceVersion := calculateNextRdmaIngressResourceVersion()
		newRdmaIngress := &objects.RdmaIngressObject{
			PortNumber:       portNumber,
			Service:          service,
			TargetPortNumber: targetPortNumber,
			ResourceVersion:  resourceVersion,
		}
		newRdmaIngressEvent := &objects.RdmaIngressEventObject{
			ResourceVersion:   resourceVersion,
			EventType:         constants.EventType_Set,
			RdmaIngressObject: *newRdmaIngress,
		}
		rdmaIngressMap[portNumber] = newRdmaIngress
		Instance().RdmaIngressEventMap[resourceVersion] = newRdmaIngressEvent
		EnqueueRdmaIngress(*newRdmaIngressEvent)

		rdmaIngressStr, _ := json.Marshal(rdmaIngressMap[portNumber])
		klog.Infof("Handling rdmaingress completed. RdmaIngress set as %s. Tracking Id: %s", rdmaIngressStr, trackingId)
	}
}

func DeleteRdmaIngress(portNumber uint16, trackingId string) {
	rdmaIngressMap := Instance().RdmaIngressMap
	rdmaIngress, exists := rdmaIngressMap[portNumber]
	if exists {
		resourceVersion := calculateNextRdmaIngressResourceVersion()
		newRdmaIngressEvent := &objects.RdmaIngressEventObject{
			ResourceVersion:   resourceVersion,
			EventType:         constants.EventType_Delete,
			RdmaIngressObject: *rdmaIngress,
		}
		Instance().RdmaIngressEventMap[resourceVersion] = newRdmaIngressEvent
		EnqueueRdmaIngress(*newRdmaIngressEvent)
		delete(rdmaIngressMap, portNumber)
		klog.Infof("Handling rdmaingress completed. RdmaIngress with port %s is deleted. Tracking Id: %s", portNumber, trackingId)
	}
}

func ListRdmaIngress(minResourceVersion int) []objects.RdmaIngressEventObject {
	maxResourceVersion := Instance().RdmaIngressResourceVersion
	rdmaIngressEventMap := Instance().RdmaIngressEventMap

	var rdmaIngressEvents []objects.RdmaIngressEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		rdmaIngressEvents = append(rdmaIngressEvents, *rdmaIngressEventMap[i])
	}
	return rdmaIngressEvents
}

func EnqueueRdmaIngress(rdmaIngressEventObject objects.RdmaIngressEventObject) {
	for _, queue := range Instance().RdmaIngressQueueMap {
		queue.Add(rdmaIngressEventObject)
	}
}

func AddRdmaIngressQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().RdmaIngressQueueMap[key] = queue
}

func RemoveRdmaIngressQueue(key uuid.UUID) {
	delete(Instance().RdmaIngressQueueMap, key)
}
