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

package state

import (
	"sync"
	"time"

	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/state/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/nat"
	natEvent "github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/session"
	sevent "github.com/mysteriumnetwork/node/session/event"
	"github.com/rs/zerolog/log"
)

// DefaultDebounceDuration is the default time interval suggested for debouncing
const DefaultDebounceDuration = time.Millisecond * 200

type natStatusProvider interface {
	Status() nat.Status
	ConsumeNATEvent(event natEvent.Event)
}

type publisher interface {
	Publish(topic string, data interface{})
}

type serviceLister interface {
	List() map[service.ID]*service.Instance
}

type serviceSessionStorage interface {
	GetAll() []session.Session
}

// Keeper keeps track of state through eventual consistency.
// This should become the de-facto place to get your info about node.
type Keeper struct {
	state                  *stateEvent.State
	lock                   sync.RWMutex
	natStatusProvider      natStatusProvider
	publisher              publisher
	serviceLister          serviceLister
	serviceSessionStorage  serviceSessionStorage
	sessionConnectionCount map[string]event.ConnectionStatistics

	ConsumeServiceStateEvent     func(e interface{})
	ConsumeNATEvent              func(e interface{})
	consumeSessionEventDebounced func(e interface{})
	announceStateChanges         func(e interface{})
}

// NewKeeper returns a new instance of the keeper
func NewKeeper(natStatusProvider natStatusProvider, publisher publisher, serviceLister serviceLister, serviceSessionStorage serviceSessionStorage, debounceDuration time.Duration) *Keeper {
	k := &Keeper{
		state: &stateEvent.State{
			NATStatus: stateEvent.NATStatus{
				Status: "not_finished",
			},
		},
		natStatusProvider:     natStatusProvider,
		publisher:             publisher,
		serviceLister:         serviceLister,
		serviceSessionStorage: serviceSessionStorage,

		sessionConnectionCount: make(map[string]event.ConnectionStatistics),
	}
	k.ConsumeServiceStateEvent = debounce(k.updateServiceState, debounceDuration)
	k.ConsumeNATEvent = debounce(k.updateNatStatus, debounceDuration)
	k.consumeSessionEventDebounced = debounce(k.updateSessionState, debounceDuration)
	k.announceStateChanges = debounce(k.announceState, debounceDuration)
	return k
}

func (k *Keeper) announceState(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.publisher.Publish(stateEvent.AppTopicState, *k.state)
}

func (k *Keeper) updateServiceState(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.updateServices()
	go k.announceStateChanges(nil)
}

// ConsumeSessionStateEvent consumes the session change events
func (k *Keeper) ConsumeSessionStateEvent(e sevent.Payload) {
	if e.Action == sevent.Acknowledged {
		k.consumeSessionAcknowledgeEvent(e)
		return
	}

	k.consumeSessionEventDebounced(e)
}

func (k *Keeper) consumeSessionAcknowledgeEvent(e sevent.Payload) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if e.Action != sevent.Acknowledged {
		return
	}
	session, found := k.getSessionByID(e.ID)
	if !found {
		return
	}

	service, found := k.getServiceByID(session.ServiceID)
	if !found {
		return
	}

	k.incrementConnectCount(service.ID, true)

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateServices() {
	services := k.serviceLister.List()
	result := make([]stateEvent.ServiceInfo, len(services))

	i := 0
	for key, v := range services {
		proposal := v.Proposal()

		// merge in the connection statistics
		match, _ := k.getServiceByID(string(key))

		result[i] = stateEvent.ServiceInfo{
			ID:                   string(key),
			ProviderID:           proposal.ProviderID,
			Type:                 proposal.ServiceType,
			Options:              v.Options(),
			Status:               string(v.State()),
			Proposal:             proposal,
			ConnectionStatistics: match.ConnectionStatistics,
		}
		i++
	}

	k.state.Services = result
}

func (k *Keeper) getServiceByID(id string) (se stateEvent.ServiceInfo, found bool) {
	for i := range k.state.Services {
		if k.state.Services[i].ID == id {
			se = k.state.Services[i]
			found = true
			return
		}
	}
	return
}

func (k *Keeper) updateNatStatus(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	event, ok := e.(natEvent.Event)
	if !ok {
		log.Warn().Msg("Received a non-NAT event on NAT status call - ignoring")
		return
	}

	k.natStatusProvider.ConsumeNATEvent(event)
	status := k.natStatusProvider.Status()
	k.state.NATStatus = stateEvent.NATStatus{Status: status.Status}
	if status.Error != nil {
		k.state.NATStatus.Error = status.Error.Error()
	}

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateSessionState(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	sessions := k.serviceSessionStorage.GetAll()
	newSessions := make([]stateEvent.ServiceSession, 0)
	result := make([]stateEvent.ServiceSession, len(sessions))
	for i := range sessions {
		result[i] = stateEvent.ServiceSession{
			ID:           string(sessions[i].ID),
			ConsumerID:   sessions[i].ConsumerID.Address,
			CreatedAt:    sessions[i].CreatedAt,
			BytesOut:     sessions[i].DataTransferred.Up,
			BytesIn:      sessions[i].DataTransferred.Down,
			TokensEarned: sessions[i].TokensEarned,
			ServiceID:    sessions[i].ServiceID,
			ServiceType:  sessions[i].ServiceType,
		}

		// each new session counts as an additional attempt, mark them for further use
		_, found := k.getSessionByID(string(result[i].ID))
		if !found {
			newSessions = append(newSessions, result[i])
		}
	}

	for i := range newSessions {
		k.incrementConnectCount(newSessions[i].ServiceID, false)
	}

	k.state.Sessions = result

	go k.announceStateChanges(nil)
}

func (k *Keeper) getSessionByID(id string) (ss stateEvent.ServiceSession, found bool) {
	for i := range k.state.Sessions {
		if k.state.Sessions[i].ID == id {
			ss = k.state.Sessions[i]
			found = true
			return
		}
	}
	return
}

func (k *Keeper) incrementConnectCount(serviceID string, isSuccess bool) {
	for i := range k.state.Services {
		if k.state.Services[i].ID == serviceID {
			if isSuccess {
				k.state.Services[i].ConnectionStatistics.Successful++
			} else {
				k.state.Services[i].ConnectionStatistics.Attempted++
			}
			break
		}
	}
}

// GetState returns the current state
func (k *Keeper) GetState() event.State {
	k.lock.Lock()
	defer k.lock.Unlock()

	return *k.state
}

// Debounce takes in the f and makes sure that it only gets called once if multiple calls are executed in the given interval d.
// It returns the debounced instance of the function.
func debounce(f func(interface{}), d time.Duration) func(interface{}) {
	incoming := make(chan interface{})

	go func() {
		var e interface{}

		t := time.NewTimer(d)
		t.Stop()

		for {
			select {
			case e = <-incoming:
				t.Reset(d)
			case <-t.C:
				go f(e)
			}
		}
	}()

	return func(e interface{}) {
		go func() { incoming <- e }()
	}
}
