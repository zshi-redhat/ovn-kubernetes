//go:build linux
// +build linux

package node

import (
	"fmt"
	"net"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/config"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/factory"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/kube"
	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/types"
	util "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/util"

	"k8s.io/klog/v2"
	utilnet "k8s.io/utils/net"
)

func newEmptyGateway(nodeName string, hostSubnets []*net.IPNet, gwNextHops []net.IP, gwIntf, egressGWIntf string, gwIPs []*net.IPNet, nodeIP string,
	nodeAnnotator kube.Annotator, kube kube.Interface, cfg *managementPortConfig, watchFactory factory.NodeWatchFactory) (*gateway, error) {
	klog.Info("Creating new empty gateway")
	gw := &gateway{}

	for _, hostSubnet := range hostSubnets {
		// empty gateway mode uses mp0 as default path for all ingress traffic into OVN
		var nextHop *net.IPNet
		if utilnet.IsIPv6CIDR(hostSubnet) {
			nextHop = cfg.ipv6.ifAddr
		} else {
			nextHop = cfg.ipv4.ifAddr
		}

		// add iptables masquerading for mp0 to exit the host for egress
		cidr := nextHop.IP.Mask(nextHop.Mask)
		cidrNet := &net.IPNet{IP: cidr, Mask: nextHop.Mask}
		err := initEmptyGatewayNATRules(types.K8sMgmtIntfName, cidrNet)
		if err != nil {
			return nil, fmt.Errorf("failed to add empty NAT rules for: %s, err: %v", types.K8sMgmtIntfName, err)
		}
	}

	chassisID, err := util.GetNodeChassisID()
	if err != nil {
		return nil, err
	}
	err = util.SetL3GatewayConfig(nodeAnnotator, &util.L3GatewayConfig{
		Mode:      config.GatewayModeDisabled,
		ChassisID: chassisID,
	})

	gw.readyFunc = func() (bool, error) { return true, nil }
	gw.initFunc = func() error {
		gw.nodeIPManager = newAddressManager(nodeName, kube, cfg, watchFactory)
		gw.nodeIPManager.nodePrimaryIP = nodeIP

		if config.Gateway.NodeportEnable {
			gw.nodePortWatcher, err = newNodePortWatcher("", "", "", nodeName, gwIPs, nil, gw.nodeIPManager, watchFactory)
			if err != nil {
				return err
			}
		}

		return nil
	}

	klog.Info("Empty Gateway Creation Complete")
	return gw, nil
}
