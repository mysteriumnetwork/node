package netstack

import (
	"net"
)

func mustParseCIDR(cidrs []string) []*net.IPNet {
	ipnets := make([]*net.IPNet, len(cidrs))
	for i, cidr := range cidrs {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		ipnets[i] = ipnet
	}
	return ipnets
}

var privateIPv4Block []*net.IPNet

var txtPrivateIPv4Block = []string{
	"127.0.0.0/8",    // IPv4 loopback
	"10.0.0.0/8",     // RFC1918
	"100.64.0.0/10",  // https://en.wikipedia.org/wiki/Reserved_IP_addresses
	"172.16.0.0/12",  // RFC1918
	"192.168.0.0/16", // RFC1918
	"169.254.0.0/16", // RFC3927 link-local
}

func init() {
	privateIPv4Block = mustParseCIDR(txtPrivateIPv4Block)
}

// isPublicAddr retruns true if the IP is a private address
func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return false
	}

	for _, block := range privateIPv4Block {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}
