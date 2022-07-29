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

package objects

import "encoding/json"

type NetVariables struct {
	Command         string
	ContainerID     string
	NetNS           string
	IfName          string
	CniPath         string
	K8sPodNamespace string
	K8sPodName      string
	CniVersion      string
	NetworkName     string
	Plugin          string
}

func (netVariables NetVariables) String() string {
	str, _ := json.Marshal(netVariables)
	return string(str)
}