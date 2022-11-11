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
	"strings"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/google/uuid"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

func calculateNextServiceResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.ServiceResourceVersion += 1
	return instance.ServiceResourceVersion
}

func SetService(name string, clusterIP string, ports []string, trackingId string) {
	serviceMap := Instance().ServiceMap
	service, exists := serviceMap[name]
	changed := false
	portsStr := strings.Join(ports, ",")
	if exists {
		if service.ClusterIP != clusterIP || service.Ports != portsStr {
			changed = true
		} else {
			klog.Infof("Handling service completed. Service %s is unchanged. Tracking Id: %s", name, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		resourceVersion := calculateNextServiceResourceVersion()
		newService := &objects.ServiceObject{
			Name:            name,
			ClusterIP:       clusterIP,
			Ports:           portsStr,
			ResourceVersion: resourceVersion,
		}
		newServiceEvent := &objects.ServiceEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Set,
			ServiceObject:   *newService,
		}
		serviceMap[name] = newService
		Instance().ServiceEventMap[resourceVersion] = newServiceEvent
		EnqueueService(*newServiceEvent)

		serviceStr, _ := json.Marshal(serviceMap[name])
		klog.Infof("Handling service completed. Service set as %s. Tracking Id: %s", serviceStr, trackingId)
	}
}

func DeleteService(name string, trackingId string) {
	serviceMap := Instance().ServiceMap
	service, exists := serviceMap[name]
	if exists {
		resourceVersion := calculateNextServiceResourceVersion()
		newServiceEvent := &objects.ServiceEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Delete,
			ServiceObject:   *service,
		}
		Instance().ServiceEventMap[resourceVersion] = newServiceEvent
		EnqueueService(*newServiceEvent)
		delete(serviceMap, name)
		klog.Infof("Handling service completed. Service %s is deleted. Tracking Id: %s", name, trackingId)
	}
}

func ListService(minResourceVersion int) []objects.ServiceEventObject {
	maxResourceVersion := Instance().ServiceResourceVersion
	serviceEventMap := Instance().ServiceEventMap

	var serviceEvents []objects.ServiceEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		serviceEvents = append(serviceEvents, *serviceEventMap[i])
	}
	return serviceEvents
}

func EnqueueService(serviceEventObject objects.ServiceEventObject) {
	for _, queue := range Instance().ServiceQueueMap {
		queue.Add(serviceEventObject)
	}
}

func AddServiceQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().ServiceQueueMap[key] = queue
}

func RemoveServiceQueue(key uuid.UUID) {
	delete(Instance().ServiceQueueMap, key)
}
