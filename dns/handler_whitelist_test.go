/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package dns

import (
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	policyDNSZone      = market.AccessPolicy{ID: "wildcard-domain"}
	policyDNSZoneRules = market.AccessPolicyRuleSet{
		ID: "wildcard-domain",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeDNSZone, Value: "wildcard.com"},
		},
	}

	policyDNSHostname      = market.AccessPolicy{ID: "domain"}
	policyDNSHostnameRules = market.AccessPolicyRuleSet{
		ID: "domain",
		Allow: []market.AccessRule{
			{Type: market.AccessPolicyTypeDNSHostname, Value: "single.com"},
		},
	}
)

func Test_WhitelistAnswers(t *testing.T) {
	tests := []struct {
		response       *dns.Msg
		whitelistedIPs map[string]int
		message        string
	}{
		{
			&dns.Msg{
				Answer: []dns.RR{
					&dns.A{
						Hdr: dns.RR_Header{Name: "single.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.3"),
					},
					&dns.A{
						Hdr: dns.RR_Header{Name: "single.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.4"),
					},
				},
			},
			map[string]int{
				"0.0.0.3": 1,
				"0.0.0.4": 1,
			},
			"should allow whitelisted hostname",
		},
		{
			&dns.Msg{
				Answer: []dns.RR{
					&dns.A{
						Hdr: dns.RR_Header{Name: "cdn.single.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.2"),
					},
				},
			},
			map[string]int{},
			"should not allow zone of whitelisted hostname",
		},
		{
			&dns.Msg{
				Answer: []dns.RR{
					&dns.A{
						Hdr: dns.RR_Header{Name: "belekas.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.1"),
					},
				},
			},
			map[string]int{},
			"should not allow unknown hostname",
		},

		{
			&dns.Msg{
				Answer: []dns.RR{
					&dns.A{
						Hdr: dns.RR_Header{Name: "wildcard.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.8"),
					},
					&dns.A{
						Hdr: dns.RR_Header{Name: "wildcard.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.9"),
					},
				},
			},
			map[string]int{
				"0.0.0.8": 1,
				"0.0.0.9": 1,
			},
			"should allow whitelisted wildcard hostname",
		},
		{
			&dns.Msg{
				Answer: []dns.RR{
					&dns.A{
						Hdr: dns.RR_Header{Name: "cdn.wildcard.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.6"),
					},
					&dns.A{
						Hdr: dns.RR_Header{Name: "cdn.wildcard.com.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
						A:   net.ParseIP("0.0.0.7"),
					},
				},
			},
			map[string]int{
				"0.0.0.6": 1,
				"0.0.0.7": 1,
			},
			"should allow zone of whitelisted wildcard hostname",
		},
	}

	for _, test := range tests {
		mockedBlocker := &trafficBlockerMock{
			allowIPCalls: map[string]int{},
		}
		writer := &recordingWriter{}
		handler := WhitelistAnswers(
			dns.HandlerFunc(func(writer dns.ResponseWriter, req *dns.Msg) {
				writer.WriteMsg(test.response)
			}),
			mockedBlocker,
			createPolicies(),
		)

		handler.ServeDNS(writer, &dns.Msg{})
		assert.Equal(t, test.whitelistedIPs, mockedBlocker.allowIPCalls, test.message)
		assert.Equal(t, test.response, writer.responseMsg)
	}
}

func createPolicies() *policy.Repository {
	repo := policy.NewRepository()
	repo.SetPolicyRules(policyDNSZone, policyDNSZoneRules)
	repo.SetPolicyRules(policyDNSHostname, policyDNSHostnameRules)
	return repo
}

type trafficBlockerMock struct {
	allowIPCalls map[string]int
}

func (tbn *trafficBlockerMock) Setup() error { return nil }

func (tbn *trafficBlockerMock) Teardown() {}

func (tbn *trafficBlockerMock) BlockIncomingTraffic(net.IPNet) (firewall.IncomingRuleRemove, error) {
	return nil, nil
}

func (tbn *trafficBlockerMock) AllowIPAccess(ip net.IP) (firewall.IncomingRuleRemove, error) {
	ipString := ip.String()
	if _, called := tbn.allowIPCalls[ipString]; !called {
		tbn.allowIPCalls[ipString] = 0
	}
	tbn.allowIPCalls[ipString]++

	return func() error {
		return nil
	}, nil
}
