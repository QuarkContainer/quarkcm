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

func calculateNextConfigMapResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.ConfigMapResourceVersion += 1
	return instance.ConfigMapResourceVersion
}

func SetConfigMap(name string, value string, trackingId string) {
	configMapMap := Instance().ConfigMapMap
	configMap, exists := configMapMap[name]
	changed := false
	if exists {
		if value != configMap.Value {
			changed = true
		} else {
			klog.Infof("Handling configMap completed. ConfigMap %s is unchanged. Tracking Id: %s", name, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		resourceVersion := calculateNextConfigMapResourceVersion()
		newConfigMap := &objects.ConfigMapObject{
			Name:            name,
			Value:           value,
			ResourceVersion: resourceVersion,
		}
		newConfigMapEvent := &objects.ConfigMapEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Set,
			ConfigMapObject: *newConfigMap,
		}
		configMapMap[name] = newConfigMap
		Instance().ConfigMapEventMap[resourceVersion] = newConfigMapEvent
		EnqueueConfigMap(*newConfigMapEvent)

		configMapStr, _ := json.Marshal(configMapMap[name])
		klog.Infof("Handling configMap completed. ConfigMap set as %s. Tracking Id: %s", configMapStr, trackingId)
	}
}

func DeleteConfigMap(name string, trackingId string) {
	configMapMap := Instance().ConfigMapMap
	configMap, exists := configMapMap[name]
	if exists {
		resourceVersion := calculateNextConfigMapResourceVersion()
		newConfigMapEvent := &objects.ConfigMapEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Delete,
			ConfigMapObject: *configMap,
		}
		Instance().ConfigMapEventMap[resourceVersion] = newConfigMapEvent
		EnqueueConfigMap(*newConfigMapEvent)
		delete(configMapMap, name)
		klog.Infof("Handling configMap completed. ConfigMap %s is deleted. Tracking Id: %s", name, trackingId)
	}
}

func ListConfigMap(minResourceVersion int) []objects.ConfigMapEventObject {
	maxResourceVersion := Instance().ConfigMapResourceVersion
	configMapEventMap := Instance().ConfigMapEventMap

	var configMapEvents []objects.ConfigMapEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		configMapEvents = append(configMapEvents, *configMapEventMap[i])
	}
	return configMapEvents
}

func EnqueueConfigMap(configMapEventObject objects.ConfigMapEventObject) {
	for _, queue := range Instance().ConfigMapQueueMap {
		queue.Add(configMapEventObject)
	}
}

func AddConfigMapQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().ConfigMapQueueMap[key] = queue
}

func RemoveConfigMapQueue(key uuid.UUID) {
	delete(Instance().ConfigMapQueueMap, key)
}
