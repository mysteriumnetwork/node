//go:build linux || darwin

package nat

import (
	"net"
	"strings"
	"testing"
)

func TestRuntimeServiceRuleUsesTCPRoutingWithoutRedirectOrUDP(t *testing.T) {
	_, vpn, err := net.ParseCIDR("10.10.0.0/24")
	if err != nil {
		t.Fatal(err)
	}
	rules := makeIPTablesRules(Options{
		VPNNetwork:     *vpn,
		ProviderExtIP:  net.ParseIP("192.0.2.1"),
		DNSIP:          net.ParseIP("10.10.0.1"),
		TCPServicePort: 3000,
	})
	var found bool
	for _, rule := range rules {
		args := strings.Join(rule.ApplyArgs(), " ")
		if strings.Contains(args, "--dport 3000") {
			found = true
			if !strings.Contains(args, "--protocol tcp") || !strings.Contains(args, "--jump RETURN") {
				t.Fatalf("runtime service traffic is not narrowly routed: %s", args)
			}
			if strings.Contains(args, "REDIRECT") || strings.Contains(args, "--protocol udp") {
				t.Fatalf("unsafe runtime service forwarding rule: %s", args)
			}
		}
	}
	if !found {
		t.Fatal("runtime TCP routing rule was not generated")
	}
}
