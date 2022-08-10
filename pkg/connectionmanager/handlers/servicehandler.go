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

	. "github.com/ahmetalpbalkan/go-linq"
)

type ServiceHandler struct {
}

func (d *ServiceHandler) Init() error {
	return nil
}

func (d *ServiceHandler) Handle(eventItem objects.EventItem) {
	klog.Infof("Handle service event %s:%s. Tracking Id: %s", eventItem.Key, eventItem.EventType, eventItem.Id)
	serviceKey := fmt.Sprintf("%s/%s", eventItem.Namespace, eventItem.Key)
	if eventItem.EventType == constants.EventType_Delete {
		datastore.DeleteService(serviceKey, eventItem.Id)
	} else {
		handleServiceSet(eventItem, serviceKey, eventItem.Obj.(*v1.Service))
	}
}

func handleServiceSet(eventItem objects.EventItem, key string, service *v1.Service) {
	datastore.SetService(key, service.Spec.ClusterIP, getPortsString(service.Spec.Ports), eventItem.Id)
}

func getPortsString(ports []v1.ServicePort) []string {
	var result []string
	From(ports).Select(func(p interface{}) interface{} {
		return fmt.Sprintf("%s:%v", p.(v1.ServicePort).Protocol, p.(v1.ServicePort).Port)
	}).ToSlice(&result)
	return result
}
