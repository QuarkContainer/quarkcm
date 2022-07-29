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

package constants

const (
	ResourceType_Pod  string = "pod"
	ResourceType_Node string = "node"

	EventType_Set    string = "set"
	EventType_Delete string = "delete"

	Pod_Cidr_File_Name_In_Container      string = "/home/PodCidr"
	Pod_Cidr_File_Name_In_Host           string = "/var/quarkcm/PodCidr"
	Allocated_Ips_File_Name_In_Container string = "/home/Allocated"
	Allocated_Ips_File_Name_In_Host      string = "/var/quarkcm/Allocated"
)
