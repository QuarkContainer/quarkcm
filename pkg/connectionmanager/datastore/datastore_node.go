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

func calculateNextNodeResourceVersion() int {
	instance := Instance()
	lock.Lock()
	defer lock.Unlock()
	instance.NodeResourceVersion += 1
	return instance.NodeResourceVersion
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
