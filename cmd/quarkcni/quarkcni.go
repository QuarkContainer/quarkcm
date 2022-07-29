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

package main

import (
	"flag"
	"runtime"

	"github.com/CentaurusInfra/quarkcm/cmd/quarkcni/app"
	"github.com/CentaurusInfra/quarkcm/pkg/objects"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	klog "k8s.io/klog/v2"
)

var netVariables objects.NetVariables

func init() {
	// Ensures runs only on main thread
	runtime.LockOSThread()

	// Initial log
	klog.InitFlags(nil)
	flag.Set("logtostderr", "false")
	flag.Set("log_file", "/var/log/quarkcni.log")
	defer klog.Flush()

	result, err := app.DoInit(&netVariables)
	if result != "" {
		klog.Infof("%s/%s CNI_INIT succeed: '%+v'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, result)
	}
	if err != nil {
		klog.Errorf("%s/%s CNI_INIT error: '%+v'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, err)
	}
}

func cmdAdd(args *skel.CmdArgs) error {
	defer klog.Flush()

	if netVariables.K8sPodName == "" {
		return nil
	}

	klog.Infof("%s/%s CNI_ADD starts: '%+v'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, &netVariables)

	result, err := app.DoCmdAdd(&netVariables, args.StdinData)
	if err != nil {
		klog.Errorf("%s/%s CNI_ADD error: '%+v'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, err)
	} else {
		klog.Infof("%s/%s CNI_ADD succeed\n", netVariables.K8sPodNamespace, netVariables.K8sPodName)
	}
	result.Print()
	return err
}

func cmdDel(args *skel.CmdArgs) error {
	defer klog.Flush()

	if netVariables.K8sPodName == "" {
		return nil
	}

	klog.Infof("%s/%s CNI_DEL starts: '%+v'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, &netVariables)

	result, err := app.DoCmdDel(&netVariables, args.StdinData)
	if err != nil {
		klog.Errorf("%s/%s CNI_DEL error: '%+v'\n", netVariables.K8sPodNamespace, netVariables.K8sPodName, err)
	} else {
		klog.Infof("%s/%s CNI_DEL succeed\n", netVariables.K8sPodNamespace, netVariables.K8sPodName)
	}
	result.Print()
	return err
}

func main() {
	defer klog.Flush()
	skel.PluginMain(cmdAdd, nil, cmdDel, version.PluginSupports("0.2.0", "0.3.0", "0.3.1"), "Quark CNI plugin")
}
