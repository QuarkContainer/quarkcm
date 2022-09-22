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
	"strings"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/datastore"
	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/objects"
	v1 "k8s.io/api/core/v1"
	"k8s.io/klog"
)

type ConfigMapHandler struct {
}

func (d *ConfigMapHandler) Handle(eventItem objects.EventItem) {
	klog.Infof("Handle configMap event %s:%s. Tracking Id: %s", eventItem.Key, eventItem.EventType, eventItem.Id)
	if eventItem.EventType == constants.EventType_Set {
		handleConfigMapSet(eventItem, eventItem.Obj.(*v1.ConfigMap))
	}
}

func handleConfigMapSet(eventItem objects.EventItem, configMap *v1.ConfigMap) {
	if configMap.ObjectMeta.Name == "kubeadm-config" {
		clusterConfiguration := configMap.Data["ClusterConfiguration"]
		podSubnet := retrieveConfig(clusterConfiguration, "podSubnet")
		if len(podSubnet) > 0 {
			datastore.SetConfigMap("podSubnet", podSubnet, eventItem.Id)
		}
		serviceSubnet := retrieveConfig(clusterConfiguration, "serviceSubnet")
		if len(serviceSubnet) > 0 {
			datastore.SetConfigMap("serviceSubnet", serviceSubnet, eventItem.Id)
		}
	}
}

func retrieveConfig(configuration string, name string) string {
	startIndex := strings.Index(configuration, name)
	if startIndex < 0 {
		return ""
	}
	subConfiguration := string(configuration[startIndex+len(name)+2:])
	endIndex := strings.Index(subConfiguration, "\n")
	if endIndex < 0 {
		return subConfiguration
	} else {
		return string(subConfiguration[:endIndex])
	}
}
