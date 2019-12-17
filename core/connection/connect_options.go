/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/utils/stringutil"
	"github.com/pkg/errors"
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

// ConnectParams holds plugin specific params
type ConnectParams struct {
	// kill switch option restricting communication only through VPN
	DisableKillSwitch bool
	// DNS servers to use
	DNS DNSOption
}

// ConnectOptions represents the params we need to ensure a successful connection
type ConnectOptions struct {
	ConsumerID    identity.Identity
	ProviderID    identity.Identity
	Proposal      market.ServiceProposal
	SessionID     session.ID
	DNS           DNSOption
	SessionConfig []byte
}
