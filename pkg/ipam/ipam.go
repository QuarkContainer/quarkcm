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

package ipam

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/CentaurusInfra/quarkcm/pkg/connectionmanager/constants"
	"github.com/CentaurusInfra/quarkcm/pkg/util/osutil"
	"k8s.io/klog"
)

var PodCidr string

func Allocate(vpc string, podName string, podId string) (string, string) {
	PodCidr = osutil.ReadFromFile(constants.Pod_Cidr_File_Name_In_Host)

	// todo Hong add a lock to avoid dirty read
	var vpcIpsMap = map[string]map[string]string{} // vpc -> ip -> pod name
	allocatedIps := osutil.ReadFromFile(constants.Allocated_Ips_File_Name_In_Host)
	json.Unmarshal([]byte(allocatedIps), &vpcIpsMap)

	ipSet, vpcExists := vpcIpsMap[vpc]
	if !vpcExists {
		ipSet = map[string]string{}
		vpcIpsMap[vpc] = ipSet
	}

	ip, ipnet, _ := net.ParseCIDR(PodCidr)
	skip := 2
	var gateway string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		if skip > 0 {
			if skip == 1 {
				gateway = ip.String()
			}
			skip -= 1
			continue
		}
		ipStr := ip.String()
		_, ipExists := ipSet[ipStr]
		if !ipExists {
			ipSet[ipStr] = fmt.Sprintf("%s|%s", podName, podId)
			vpcIpsJson, _ := json.Marshal(vpcIpsMap)
			osutil.WriteToFile(constants.Allocated_Ips_File_Name_In_Host, string(vpcIpsJson))
			klog.Infof("Allocated ip %s gateway %s for vpc %s pod %s pod id %s", ipStr, gateway, vpc, podName, podId)
			return ipStr, gateway
		}
	}

	return "", ""
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func Deallocate(vpc string, podName string) {
	// todo Hong add a lock to avoid dirty read
	// todo Hong If vpc has no pod, delete vpc
	var vpcIpsMap = map[string]map[string]string{}
	allocatedIps := osutil.ReadFromFile(constants.Allocated_Ips_File_Name_In_Host)
	json.Unmarshal([]byte(allocatedIps), &vpcIpsMap)

	ipSet, vpcExists := vpcIpsMap[vpc]
	if !vpcExists {
		return
	}

	for key, value := range ipSet {
		if strings.Split(value, "|")[0] == podName {
			delete(ipSet, key)
			break
		}
	}

	vpcIpsJson, _ := json.Marshal(vpcIpsMap)
	osutil.WriteToFile(constants.Allocated_Ips_File_Name_In_Host, string(vpcIpsJson))
}
