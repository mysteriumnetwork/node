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

package nat

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/node"

	"github.com/mysteriumnetwork/node/core"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/identity"
)

// NATStatusV2 enum
// TODO: V2 suffix should be removed once previous tracker is removed
type NATStatusV2 string

const (
	// Passed enum
	Passed NATStatusV2 = "passed"
	// Failed enum
	Failed NATStatusV2 = "failed"
	// Pending enum
	Pending NATStatusV2 = "pending"
)

// AppTopicNATStatusUpdate nat status update topic for event bus
const AppTopicNATStatusUpdate = "AppTopicNATStatusUpdate"

type currentIdentity interface {
	GetUnlockedIdentity() (identity.Identity, bool)
}

type publisher interface {
	Publish(topic string, data interface{})
}

// ProviderSessions should return provider session monitoring state
type ProviderSessions func(providerID string) []Session

// Session represent session monitoring state
type Session struct {
	ProviderID       string
	ServiceType      string
	MonitoringFailed bool
}

// V2NatStatusEvent nat status event
type V2NatStatusEvent struct {
	Status NATStatusV2
}

// StatusTrackerV2 tracks NAT status for service
type StatusTrackerV2 struct {
	lock   sync.Mutex
	status NATStatusV2

	publisher publisher

	providerSessions ProviderSessions
	currentIdentity  currentIdentity

	pollInterval time.Duration
}

// NewStatusTrackerV2 constructor
func NewStatusTrackerV2(
	providerSessions ProviderSessions,
	currentIdentity currentIdentity,
	publisher publisher,
	options node.OptionsNATStatusTrackerV2,
) *StatusTrackerV2 {
	validatedInterval := options.PollInterval
	if validatedInterval < time.Minute {
		validatedInterval = time.Minute
	}
	keeper := &StatusTrackerV2{
		providerSessions: providerSessions,
		currentIdentity:  currentIdentity,
		publisher:        publisher,
		pollInterval:     validatedInterval,
	}
	// consumers don't need service NAT status
	if !config.GetBool(config.FlagConsumer) {
		keeper.startPolling()
	}
	return keeper
}

func (k *StatusTrackerV2) startPolling() {
	go func() {
		for {
			k.updateAndAnnounce()
			time.Sleep(k.pollInterval)
		}
	}()
}

func (k *StatusTrackerV2) updateAndAnnounce() {
	k.lock.Lock()
	defer k.lock.Unlock()
	id, ok := k.currentIdentity.GetUnlockedIdentity()
	prevStatus := k.status
	if ok {
		k.status = resolveNATStatus(k.providerSessions(id.Address))
	} else {
		k.status = Pending
	}
	if prevStatus != k.status {
		k.publisher.Publish(AppTopicNATStatusUpdate, V2NatStatusEvent{Status: k.status})
	}
}

// StatusForce force update status and return current result
func (k *StatusTrackerV2) StatusForce() NATStatusV2 {
	k.updateAndAnnounce()
	return k.status
}

// Status return current status
func (k *StatusTrackerV2) Status() NATStatusV2 {
	k.lock.Lock()
	defer k.lock.Unlock()
	return k.status
}

func resolveNATStatus(sessions []Session) NATStatusV2 {
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
