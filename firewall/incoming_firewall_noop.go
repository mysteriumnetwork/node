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

package firewall

import (
	"net"

	"github.com/rs/zerolog/log"
)

// incomingFirewallNoop is a implementation which only logs allow requests with no effects.
// Used by default.
type incomingFirewallNoop struct{}

// Setup noop setup (just log call).
func (ifn *incomingFirewallNoop) Setup() error {
	log.Info().Msg("Rules bootstrap was requested")
	return nil
}

// Teardown noop cleanup (just log call).
func (ifn *incomingFirewallNoop) Teardown() {
	log.Info().Msg("Rules reset was requested")
}

// BlockOutgoingTraffic just logs the call.
func (ifn *incomingFirewallNoop) BlockIncomingTraffic(network net.IPNet) (IncomingRuleRemove, error) {
	log.Info().Msg("Incoming traffic block requested")
	return func() error {
		log.Info().Msg("Incoming traffic block removed")
		return nil
	}, nil
}

// AllowIPAccess logs URL for which access was requested.
func (ifn *incomingFirewallNoop) AllowURLAccess(rawURLs ...string) (IncomingRuleRemove, error) {
	for _, rawURL := range rawURLs {
		log.Info().Msgf("Allow URL %s access", rawURL)
	}
	return func() error {
		for _, rawURL := range rawURLs {
			log.Info().Msgf("Rule for URL: %s removed", rawURL)
		}
		return nil
	}, nil
}

// AllowIPAccess logs IP for which access was requested.
func (ifn *incomingFirewallNoop) AllowIPAccess(ip net.IP) (IncomingRuleRemove, error) {
	log.Info().Msgf("Allow IP %s access", ip)
	return func() error {
		log.Info().Msgf("Rule for IP: %s removed", ip)
		return nil
	}, nil
}

var _ IncomingTrafficFirewall = &incomingFirewallNoop{}
