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

// noopTrafficBlocker is a Vendor implementation which only logs allow requests with no effects
// used by default
type noopTrafficBlocker struct{}

// Setup noop setup (just log call)
func (ntb *noopTrafficBlocker) Setup() error {
	return nil
}

// Teardown noop cleanup (just log call)
func (ntb *noopTrafficBlocker) Teardown() {
	log.Info().Msg("Rules reset was requested")
}

// BlockOutgoingTraffic just logs the call
func (ntb *noopTrafficBlocker) BlockOutgoingTraffic(scope Scope, outboundIP string) (RemoveRule, error) {
	log.Info().Msg("Outgoing traffic block requested")
	return func() {
		log.Info().Msg("Outgoing traffic block removed")
	}, nil
}

// AllowIPAccess logs IP for which access was requested
func (ntb *noopTrafficBlocker) AllowIPAccess(ip string) (RemoveRule, error) {
	log.Info().Msgf("Allow IP %s access", ip)
	return func() {
		log.Info().Msgf("Rule for IP: %s removed", ip)
	}, nil
}

// AllowIPAccess logs URL for which access was requested
func (ntb *noopTrafficBlocker) AllowURLAccess(rawURLs ...string) (RemoveRule, error) {
	for _, rawURL := range rawURLs {
		log.Info().Msgf("Allow URL %s access", rawURL)
	}
	return func() {
		for _, rawURL := range rawURLs {
			log.Info().Msgf("Rule for URL: %s removed", rawURL)
		}
	}, nil
}

var _ TrafficBlocker = &noopTrafficBlocker{}
