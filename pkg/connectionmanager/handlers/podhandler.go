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

package handlers

import (
	"fmt"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/datastore"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type PodHandler struct {
}

func (d *PodHandler) Init() error {
	return nil
}

func (d *PodHandler) Handle(eventItem objects.EventItem) {
	klog.Infof("Handle pod event %s:%s. Tracking Id: %s", eventItem.Key, eventItem.EventType, eventItem.Id)
	podKey := fmt.Sprintf("%s/%s", eventItem.Namespace, eventItem.Key)
	if eventItem.EventType == constants.EventType_Delete {
		datastore.DeletePod(podKey, eventItem.Id)
	} else {
		handlePodSet(eventItem, podKey, eventItem.Obj.(*v1.Pod))
	}
}

func handlePodSet(eventItem objects.EventItem, key string, pod *v1.Pod) {
	if len(pod.Spec.NodeName) == 0 || len(pod.Status.PodIP) == 0 {
		klog.Infof("Handling pod completed. Pod %s is not ready. Tracking Id: %s", eventItem.Key, eventItem.Id)
		return
	}
	datastore.SetPod(key, pod.Status.PodIP, pod.Spec.NodeName, eventItem.Id)
}
