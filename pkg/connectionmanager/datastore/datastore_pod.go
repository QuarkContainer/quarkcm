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
	"github.com/CentaurusInfra/quarkcm/pkg/util/osutil"
	"github.com/google/uuid"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

func calculateNextPodResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.PodResourceVersion += 1
	return instance.PodResourceVersion
}

func SetPod(key string, podIP string, nodeName string, trackingId string) {
	_, isNodeIP := Instance().NodeIPMap[podIP]
	if isNodeIP {
		klog.Infof("IP of pod %s is the same as node's ip. Ignore the pod. Tracking Id: %s", key, trackingId)
		return
	}

	podMap := Instance().PodMap
	pod, exists := podMap[key]
	changed := false
	if exists {
		if pod.IP != podIP || pod.NodeName != nodeName {
			changed = true
		} else {
			klog.Infof("Handling pod completed. Pod %s is unchanged. Tracking Id: %s", key, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		containerId := GetContainerId(key, podIP)
		if containerId == "" {
			klog.Infof("Handling pod completed. Pod %s is not assigned in this node. Tracking Id: %s", key, trackingId)
			return
		}

		resourceVersion := calculateNextPodResourceVersion()
		newPod := &objects.PodObject{
			Key:             key,
			IP:              podIP,
			NodeName:        nodeName,
			ContainerID:     containerId,
			ResourceVersion: resourceVersion,
		}
		newPodEvent := &objects.PodEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Set,
			PodObject:       *newPod,
		}
		podMap[key] = newPod
		Instance().PodEventMap[resourceVersion] = newPodEvent
		EnqueuePod(*newPodEvent)

		podStr, _ := json.Marshal(podMap[key])
		klog.Infof("Handling pod completed. Pod set as %s. Tracking Id: %s", podStr, trackingId)
	}
}

func GetContainerId(podKey string, podId string) string {
	var vpcIpsMap = map[string]map[string]string{}
	allocatedIps := osutil.ReadFromFile(constants.Allocated_Ips_File_Name_In_Container)
	if allocatedIps == "" {
		return ""
	}
	json.Unmarshal([]byte(allocatedIps), &vpcIpsMap)
	namespace := strings.Split(podKey, "/")[0]
	if ipMap, exists := vpcIpsMap[namespace]; !exists {
		return ""
	} else if podStr, exists := ipMap[podId]; !exists {
		return ""
	} else {
		return strings.Split(podStr, "|")[1]
	}
}

func DeletePod(key string, trackingId string) {
	podMap := Instance().PodMap
	pod, exists := podMap[key]
	if exists {
		resourceVersion := calculateNextPodResourceVersion()
		newPodEvent := &objects.PodEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Delete,
			PodObject:       *pod,
		}
		Instance().PodEventMap[resourceVersion] = newPodEvent
		EnqueuePod(*newPodEvent)
		delete(podMap, key)
		klog.Infof("Handling pod completed. Pod %s is deleted. Tracking Id: %s", key, trackingId)
	}
}

func ListPod(minResourceVersion int) []objects.PodEventObject {
	maxResourceVersion := Instance().PodResourceVersion
	podEventMap := Instance().PodEventMap

	var podEvents []objects.PodEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		podEvents = append(podEvents, *podEventMap[i])
	}
	return podEvents
}

func EnqueuePod(podEventObject objects.PodEventObject) {
	for _, queue := range Instance().PodQueueMap {
		queue.Add(podEventObject)
	}
}

func AddPodQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().PodQueueMap[key] = queue
}

func RemovePodQueue(key uuid.UUID) {
	delete(Instance().PodQueueMap, key)
}
