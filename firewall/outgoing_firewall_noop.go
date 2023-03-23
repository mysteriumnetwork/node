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
	"github.com/rs/zerolog/log"
)

// outgoingFirewallNoop is a Vendor implementation which only logs allow requests with no effects.
// Used by default.
type outgoingFirewallNoop struct{}

// Setup noop setup (just log call).
func (ofn *outgoingFirewallNoop) Setup() error {
	return nil
}

// Teardown noop cleanup (just log call).
func (ofn *outgoingFirewallNoop) Teardown() {
	log.Info().Msg("Rules reset was requested")
}

// BlockOutgoingTraffic just logs the call.
func (ofn *outgoingFirewallNoop) BlockOutgoingTraffic(scope Scope, outboundIP string) (OutgoingRuleRemove, error) {
	log.Info().Msg("Outgoing traffic block requested")
	return func() {
		log.Info().Msg("Outgoing traffic block removed")
	}, nil
}

// AllowIPAccess logs IP for which access was requested.
func (ofn *outgoingFirewallNoop) AllowIPAccess(ip string) (OutgoingRuleRemove, error) {
	log.Info().Msg("Allow IP access")
	return func() {
		log.Info().Msg("Rule for IP removed")
	}, nil
}

// AllowIPAccess logs URL for which access was requested.
func (ofn *outgoingFirewallNoop) AllowURLAccess(rawURLs ...string) (OutgoingRuleRemove, error) {
	for _, rawURL := range rawURLs {
		log.Info().Msgf("Allow URL %s access", rawURL)
	}
	return func() {
		for _, rawURL := range rawURLs {
			log.Debug().Msgf("Rule for URL: %s removed", rawURL)
		}
	}, nil
}

var _ OutgoingTrafficFirewall = &outgoingFirewallNoop{}
