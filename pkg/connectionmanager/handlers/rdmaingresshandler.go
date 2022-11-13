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
	v1alpha1 "github.com/CentaurusInfra/quarkcm/pkg/apis/rdmaingresscontroller/v1alpha1"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/datastore"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	"k8s.io/klog"
)

type RdmaIngressHandler struct {
}

func (d *RdmaIngressHandler) Handle(eventItem objects.EventItem) {
	klog.Infof("Handle rdmaingress event %s:%s. Tracking Id: %s", eventItem.Key, eventItem.EventType, eventItem.Id)
	rdmaIngress := eventItem.Obj.(*v1alpha1.RdmaIngress)
	for _, rule := range rdmaIngress.Spec.Rules {
		port := uint16(rule.Port)
		if eventItem.EventType == constants.EventType_Delete {
			datastore.DeleteRdmaIngress(port, eventItem.Id)
		} else {
			datastore.SetRdmaIngress(port, rule.Service, uint16(rule.TargetPort), eventItem.Id)
		}
	}

}
