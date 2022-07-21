/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package node

import (
	"github.com/mysteriumnetwork/node/core"

	"github.com/mysteriumnetwork/node/identity"
)

// MonitoringStatus enum
type MonitoringStatus string

const (
	// Passed enum
	Passed MonitoringStatus = "passed"
	// Failed enum
	Failed MonitoringStatus = "failed"
	// Pending enum
	Pending MonitoringStatus = "pending"
)

type currentIdentity interface {
	GetUnlockedIdentity() (identity.Identity, bool)
}

// ProviderSessions should return provider session monitoring state
type ProviderSessions func(providerID string) []Session

// Session represent session monitoring state
type Session struct {
	ProviderID       string
	ServiceType      string
	MonitoringFailed bool
}

// MonitoringStatusTracker tracks node status for service
type MonitoringStatusTracker struct {
	providerSessions ProviderSessions
	currentIdentity  currentIdentity
}

// NewMonitoringStatusTracker constructor
func NewMonitoringStatusTracker(
	providerSessions ProviderSessions,
	currentIdentity currentIdentity,
) *MonitoringStatusTracker {
	keeper := &MonitoringStatusTracker{
		providerSessions: providerSessions,
		currentIdentity:  currentIdentity,
	}
	return keeper
}

// Status retrieves and resolved monitoring status from quality oracle
func (k *MonitoringStatusTracker) Status() MonitoringStatus {
	id, ok := k.currentIdentity.GetUnlockedIdentity()

	if ok {
		return resolveMonitoringStatus(k.providerSessions(id.Address))
	}

	return Pending
}

// MonitoringAgentStatuses enum
type MonitoringAgentStatuses map[string]map[string]int

type ProviderStatuses func(providerID string) map[string]map[string]int

type MonitoringAgentTracker struct {
	providerStatuses ProviderStatuses
	currentIdentity  currentIdentity
}

// NewMonitoringAgentTracker constructor
func NewMonitoringAgentTracker(
	providerStatuses ProviderStatuses,
	currentIdentity currentIdentity,
) *MonitoringAgentTracker {
	mat := &MonitoringAgentTracker{
		providerStatuses: providerStatuses,
		currentIdentity:  currentIdentity,
	}
	return mat
}

// Statuses retrieves and resolved monitoring status from quality oracle
func (m *MonitoringAgentTracker) Statuses() string {

	id, ok := m.currentIdentity.GetUnlockedIdentity()

	if ok {
		return resolveMonitoringAgentStatuses(m.providerStatuses(id.Address))
	}

	return "SUPER"
}

func resolveMonitoringAgentStatuses(statuses map[string]map[string]int) string {
	return "SUPER-SUPER"
}

func resolveMonitoringStatus(sessions []Session) MonitoringStatus {
	wgSession, ok := findWireGuard(sessions)
	if !ok {
		return Pending
	}

	if wgSession.MonitoringFailed {
		return Failed
	}
	return Passed
}

// openvpn is considered deprecated
func findWireGuard(sessions []Session) (Session, bool) {
	for _, s := range sessions {
		if s.ServiceType == core.WireGuard.String() {
			return s, true
		}
	}
	return Session{}, false
}
