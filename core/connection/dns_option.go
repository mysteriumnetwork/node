/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/mysteriumnetwork/node/dns"
	"github.com/mysteriumnetwork/node/utils/stringutil"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// DNSOption defines DNS server selection strategy for consumer
type DNSOption string

const (
	// DNSOptionAuto (default) tries the following with fallbacks: provider's DNS -> client's system DNS -> public DNS
	DNSOptionAuto = DNSOption("auto")
	// DNSOptionProvider uses DNS servers from provider's system configuration
	DNSOptionProvider = DNSOption("provider")
	// DNSOptionSystem uses DNS servers from client's system configuration
	DNSOptionSystem = DNSOption("system")
)

// NewDNSOption creates and validates DNSOption
func NewDNSOption(str string) (DNSOption, error) {
	opt := DNSOption(str)
	switch opt {
	case DNSOptionAuto, DNSOptionProvider, DNSOptionSystem, "":
		return opt, nil
	}
	// It may also be a set of IP addresses, e.g. 1.1.1.1,8.8.8.8
	split := strings.Split(str, ",")
	for _, s := range split {
		if ip := net.ParseIP(s); ip == nil {
			return "", errors.New("invalid IP address provided as a DNS option: " + s)
		}
	}
	return opt, nil
}

// UnmarshalJSON parses JSON → DNSOption
func (o *DNSOption) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	option, err := NewDNSOption(str)
	if err != nil {
		return err
	}
	*o = option
	return nil
}

// Exact returns a slice of DNS server IPs, if they were set
func (o DNSOption) Exact() (servers []string, ok bool) {
	switch o {
	case DNSOptionAuto, DNSOptionProvider, DNSOptionSystem:
		return nil, false
	}
	return stringutil.Split(string(o), ','), true
}

// ResolveIPs resolves DNS server IPs on the consumer side using self as the
// consumer preference and `providerDNS` argument as received from the provider
func (o *DNSOption) ResolveIPs(providerDNS string) ([]string, error) {
	log.Debug().Msg("Selecting DNS servers using strategy: " + string(*o))
	if exact, ok := o.Exact(); ok {
		return exact, nil
	}
	switch *o {
	case DNSOptionProvider:
		return selectProviderDNS(providerDNS)
	case DNSOptionSystem:
		return selectSystemDNS()
	case DNSOptionAuto:
		log.Debug().Msg("Attempting to use provider DNS")
		if providerDNS, err := selectProviderDNS(providerDNS); err == nil {
			return providerDNS, nil
		}
		log.Debug().Msg("Attempting to use system DNS")
		if systemDNS, err := selectSystemDNS(); err == nil {
			return systemDNS, nil
		}
	}
	log.Debug().Msg("Falling back to public DNS")
	return []string{"1.1.1.1", "8.8.8.8"}, nil
}

func selectSystemDNS() ([]string, error) {
	systemDNS, err := dns.ConfiguredServers()
	return systemDNS, errors.Wrap(err, "system DNS is not available")
}

func selectProviderDNS(providerDNS string) ([]string, error) {
	opt, err := NewDNSOption(providerDNS)
	if err != nil {
		return nil, errors.Wrap(err, "can't parse provider DNS string")
	}
	servers, ok := opt.Exact()
	if !ok || len(servers) == 0 {
		return nil, errors.New("provider DNS is not available")
	}
	return servers, nil
}
