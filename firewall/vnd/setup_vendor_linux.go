//+build !android

package vnd

import (
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/firewall/iptables"
)

func SetupVendor() (*iptables.Iptables, error) {
	ip, err := ip.GetOutbound()
	if err != nil {
		return nil, err
	}
	iptables := iptables.New(ip)
	if err := iptables.Setup(); err != nil {
		return nil, err
	}
	return iptables, nil
}
