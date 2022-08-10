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

type EndpointsHandler struct {
}

func (d *EndpointsHandler) Init() error {
	return nil
}

func (d *EndpointsHandler) Handle(eventItem objects.EventItem) {
	klog.Infof("Handle endpoints event %s:%s. Tracking Id: %s", eventItem.Key, eventItem.EventType, eventItem.Id)
	endpointsKey := fmt.Sprintf("%s/%s", eventItem.Namespace, eventItem.Key)
	if eventItem.EventType == constants.EventType_Delete {
		datastore.DeleteEndpoints(endpointsKey, eventItem.Id)
	} else {
		handleEndpointsSet(eventItem, endpointsKey, eventItem.Obj.(*v1.Endpoints))
	}
}

func handleEndpointsSet(eventItem objects.EventItem, key string, endpoints *v1.Endpoints) {
	datastore.SetEndpoints(key, getIPWithPortsString(endpoints.Subsets), eventItem.Id)
}

func getIPWithPortsString(subsets []v1.EndpointSubset) []string {
	var result []string
	From(subsets).ForEach(func(subset interface{}) {
		From(subset.(v1.EndpointSubset).Addresses).ForEach(func(address interface{}) {
			From(subset.(v1.EndpointSubset).Ports).ForEach(func(port interface{}) {
				result = append(result, fmt.Sprintf("%s:%s:%v", address.(v1.EndpointAddress).IP, port.(v1.EndpointPort).Protocol, port.(v1.EndpointPort).Port))
			})
		})
	})
	return result
}
