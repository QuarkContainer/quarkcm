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
	"sync"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"github.com/CentaurusInfra/quarkcm/pkg/util/osutil"
	"github.com/google/uuid"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
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
			}
		}
	}
	return dataStore
}

func calculateNextNodeResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.NodeResourceVersion += 1
	return instance.NodeResourceVersion
}

func calculateNextPodResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.PodResourceVersion += 1
	return instance.PodResourceVersion
}

func SetNode(name string, nodeHostname string, nodeIP string, creationTimestamp uint64, subnet uint32, netMask uint32, trackingId string) {
	nodeMap := Instance().NodeMap
	node, exists := nodeMap[name]
	changed := false
	if exists {
		if node.Hostname != nodeHostname || node.IP != nodeIP || node.Subnet != subnet || node.NetMask != netMask || node.CreationTimestamp != creationTimestamp {
			changed = true
		} else {
			klog.Infof("Handling node completed. Node %s is unchanged. Tracking Id: %s", name, trackingId)
		}
	} else {
		changed = true
	}

	if changed {
		resourceVersion := calculateNextNodeResourceVersion()
		newNode := &objects.NodeObject{
			Name:              name,
			Hostname:          nodeHostname,
			IP:                nodeIP,
			CreationTimestamp: creationTimestamp,
			Subnet:            subnet,
			NetMask:           netMask,
			ResourceVersion:   resourceVersion,
		}
		newNodeEvent := &objects.NodeEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Set,
			NodeObject:      *newNode,
		}
		nodeMap[name] = newNode
		Instance().NodeEventMap[resourceVersion] = newNodeEvent
		EnqueueNode(*newNodeEvent)

		Instance().NodeIPMap[nodeIP] = true

		nodeStr, _ := json.Marshal(nodeMap[name])
		klog.Infof("Handling node completed. Node set as %s. Tracking Id: %s", nodeStr, trackingId)
	}
}

func DeleteNode(name string, trackingId string) {
	nodeMap := Instance().NodeMap
	node, exists := nodeMap[name]
	if exists {
		delete(Instance().NodeIPMap, node.IP)
		resourceVersion := calculateNextNodeResourceVersion()
		newNodeEvent := &objects.NodeEventObject{
			ResourceVersion: resourceVersion,
			EventType:       constants.EventType_Delete,
			NodeObject:      *node,
		}
		Instance().NodeEventMap[resourceVersion] = newNodeEvent
		EnqueueNode(*newNodeEvent)
		delete(nodeMap, name)
		klog.Infof("Handling node completed. Node %s is deleted. Tracking Id: %s", name, trackingId)
	}
}

func ListNode(minResourceVersion int) []objects.NodeEventObject {
	maxResourceVersion := Instance().NodeResourceVersion
	nodeEventMap := Instance().NodeEventMap

	var nodeEvents []objects.NodeEventObject
	for i := minResourceVersion + 1; i <= maxResourceVersion; i++ {
		nodeEvents = append(nodeEvents, *nodeEventMap[i])
	}
	return nodeEvents
}

func EnqueueNode(nodeEventObject objects.NodeEventObject) {
	for _, queue := range Instance().NodeQueueMap {
		queue.Add(nodeEventObject)
	}
}

func AddNodeQueue(key uuid.UUID, queue workqueue.RateLimitingInterface) {
	Instance().NodeQueueMap[key] = queue
}

func RemoveNodeQueue(key uuid.UUID) {
	delete(Instance().NodeQueueMap, key)
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
