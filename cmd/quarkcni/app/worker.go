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

package app

import (
	"github.com/CentaurusInfra/quarkcm/pkg/ipam"
	"github.com/CentaurusInfra/quarkcm/pkg/objects"
	"github.com/CentaurusInfra/quarkcm/pkg/util/netutil"
	"github.com/CentaurusInfra/quarkcm/pkg/util/netvariablesutil"

	cniTypesVer "github.com/containernetworking/cni/pkg/types/current"
	klog "k8s.io/klog/v2"
)

func DoInit(netVariables *objects.NetVariables) (string, error) {
	netvariablesutil.LoadEnvVariables(netVariables)
	return netvariablesutil.MountNetNSIfNeeded(netVariables)
}

func DoCmdAdd(netVariables *objects.NetVariables, stdinData []byte) (cniTypesVer.Result, error) {
	if err := netvariablesutil.LoadCniConfig(netVariables, stdinData); err != nil {
		return cniTypesVer.Result{CNIVersion: netVariables.CniVersion}, err
	}

	allocatedIp, gateway := ipam.Allocate(netVariables.K8sPodNamespace, netVariables.K8sPodName, netVariables.ContainerID)
	klog.Infof("%s/%s allocated ip %s gateway %s\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, allocatedIp, gateway)
	_, podIP, _ := netutil.ParseCIDR(allocatedIp)

	err := netutil.ActivateInterface(netVariables.IfName, netVariables.NetNS, "32", allocatedIp, gateway)
	if err != nil {
		return cniTypesVer.Result{CNIVersion: netVariables.CniVersion}, err
	}

	result := cniTypesVer.Result{
		CNIVersion: netVariables.CniVersion,
		IPs: []*cniTypesVer.IPConfig{
			{
				Version: "4",
				Address: *podIP,
				Gateway: netutil.ParseIP(gateway),
			},
		},
	}

	return result, nil
}

func DoCmdDel(netVariables *objects.NetVariables, stdinData []byte) (cniTypesVer.Result, error) {
	result := cniTypesVer.Result{
		CNIVersion: netVariables.CniVersion,
	}

	ipam.Deallocate(netVariables.K8sPodNamespace, netVariables.K8sPodName)

	if err := netvariablesutil.LoadCniConfig(netVariables, stdinData); err != nil {
		return result, err
	}
	klog.Infof("%s/%s CNI_DEL: Deleting NetNS: '%s'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, netVariables.NetNS)
	netutil.DeleteNetNS(netVariables.NetNS)

	return result, nil
}
