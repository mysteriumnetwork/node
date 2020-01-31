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

// NewIncomingTrafficBlockerNoop creates instance of noop traffic blocker
func NewIncomingTrafficBlockerNoop() IncomingTrafficBlocker {
	return &incomingBlockerNoop{}
}

// incomingBlockerNoop is a implementation which only logs allow requests with no effects
// used by default
type incomingBlockerNoop struct{}

// Setup noop setup (just log call)
func (ibn *incomingBlockerNoop) Setup() error {
	log.Info().Msg("Rules bootstrap was requested")
	return nil
}

// Teardown noop cleanup (just log call)
func (ibn *incomingBlockerNoop) Teardown() {
	log.Info().Msg("Rules reset was requested")
}

// BlockOutgoingTraffic just logs the call
func (ibn *incomingBlockerNoop) BlockIncomingTraffic(network net.IPNet) (IncomingRuleRemove, error) {
	log.Info().Msg("Incoming traffic block requested")
	return func() error {
		log.Info().Msg("Incoming traffic block removed")
		return nil
	}, nil
}

// AllowIPAccess logs IP for which access was requested
func (ibn *incomingBlockerNoop) AllowIPAccess(ip net.IP) (IncomingRuleRemove, error) {
	log.Info().Msgf("Allow IP %s access", ip)
	return func() error {
		log.Info().Msgf("Rule for IP: %s removed", ip)
		return nil
	}, nil
}

var _ IncomingTrafficBlocker = &incomingBlockerNoop{}
