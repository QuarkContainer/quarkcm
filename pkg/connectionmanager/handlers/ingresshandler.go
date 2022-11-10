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
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/datastore"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/klog"
)

type IngressHandler struct {
}

func (d *IngressHandler) Handle(eventItem objects.EventItem) {
	klog.Infof("Handle ingress event %s:%s. Tracking Id: %s", eventItem.Key, eventItem.EventType, eventItem.Id)
	if eventItem.EventType == constants.EventType_Delete {
		datastore.DeleteIngress(eventItem.Key, eventItem.Id)
	} else {
		handleIngressSet(eventItem, eventItem.Obj.(*v1.Ingress))
	}
}

func handleIngressSet(eventItem objects.EventItem, ingress *v1.Ingress) {
	if ingress.Spec.DefaultBackend != nil {
		service := ingress.Spec.DefaultBackend.Service.Name
		portNumber := ingress.Spec.DefaultBackend.Service.Port.Number
		datastore.SetIngress(eventItem.Key, service, uint16(portNumber), eventItem.Id)
	}
}
