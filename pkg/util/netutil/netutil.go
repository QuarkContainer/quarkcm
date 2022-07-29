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

package netutil

import (
	"crypto/rand"
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"

	"github.com/CentaurusInfra/quarkcm/pkg/util/executil"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"
	"k8s.io/klog"
)

// moves the interface to the CNI netns, renames it, sets the IP address and gatewey.
func ActivateInterface(
	ifName string,
	netNSName string,
	ipPrefix string,
	ipAddress string,
	gatewayIp string) error {

	vethName, vethPeer := randomVethNameAndPeer()
	klog.Infof("Creating interface '%s'", vethName)
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: vethName,
			MTU:  9000,
		},
		PeerName: vethPeer,
	}
	err := netlink.LinkAdd(veth)
	if err != nil {
		klog.Errorf("Failed to create interface '%s'", vethName)
		return err
	}

	link, err := netlink.LinkByName(vethName)
	if err != nil {
		klog.Errorf("Failed to get interface '%s'", vethName)
		return err
	}

	if link.Attrs().OperState == netlink.OperUp {
		klog.Errorf("Interface '%s' already UP.", vethName)
		return nil
	}

	netNS, err := ns.GetNS(netNSName)
	if err != nil {
		klog.Errorf("Failed to open netns '%s'", netNSName)
		return err
	}
	defer netNS.Close()

	_, netNSFileName := path.Split(netNSName)

	klog.Infof("Move interface '%s/%d' to netns '%s'", vethName, link.Attrs().Index, netNSFileName)
	if netlink.LinkSetNsFd(link, int(netNS.Fd())); err != nil {
		return err
	}

	if err = netNS.Do(func(_ ns.NetNS) error {
		klog.Infof("Rename interface '%s' to '%s'", vethName, ifName)
		if netlink.LinkSetName(link, ifName); err != nil {
			return err
		}

		klog.Info("Retrieve loopback interface")
		loLink, err := netlink.LinkByName("lo")
		if err != nil {
			return err
		}

		klog.Info("Set loopback interface UP")
		if err = netlink.LinkSetUp(loLink); err != nil {
			return err
		}

		klog.Infof("Set interface '%s' UP", ifName)
		if err = netlink.LinkSetUp(link); err != nil {
			return err
		}

		klog.Infof("Set ip addr '%s' on interface '%s'", ipAddress, ifName)
		ipPrefix, err := strconv.Atoi(ipPrefix)
		if err != nil {
			return err
		}
		ipConfig := &netlink.Addr{
			IPNet: &net.IPNet{
				IP:   net.ParseIP(ipAddress),
				Mask: net.CIDRMask(ipPrefix, 32),
			}}
		if err = netlink.AddrAdd(link, ipConfig); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func randomVethNameAndPeer() (string, string) {
	entropy := make([]byte, 4)
	rand.Read(entropy)

	return fmt.Sprintf("eth-%x", entropy), fmt.Sprintf("veth-%x", entropy)
}

func DeleteNetNS(netNSName string) {
	netNS, _ := ns.GetNS(netNSName)
	if netNS != nil {
		netNS.Close()
	}
	_, netNSFileName := path.Split(netNSName)
	executil.Execute("ip", "netns", "delete", netNSFileName)
}

func ParseCIDR(s string) (net.IP, *net.IPNet, error) {
	if !strings.Contains(s, "/") {
		s = fmt.Sprintf("%s/%d", s, 32)
	}
	return net.ParseCIDR(s)
}

func ParseIP(s string) net.IP {
	return net.ParseIP(s)
}
