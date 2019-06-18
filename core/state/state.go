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
	state                 *stateEvent.State
	lock                  sync.RWMutex
	natStatusProvider     natStatusProvider
	publisher             publisher
	serviceLister         serviceLister
	serviceSessionStorage serviceSessionStorage

	ConsumeServiceStateEvent func(e interface{})
	ConsumeNATEvent          func(e interface{})
	ConsumeSessionEvent      func(e interface{})
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
	}
	k.ConsumeServiceStateEvent = debounce(k.updateServiceState, debounceDuration)
	k.ConsumeNATEvent = debounce(k.updateNatStatus, debounceDuration)
	k.ConsumeSessionEvent = debounce(k.updateSessionState, debounceDuration)
	return k
}

func (k *Keeper) updateServiceState(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.updateServices()
	go k.publisher.Publish(stateEvent.Topic, *k.state)
}

func (k *Keeper) updateServices() {
	services := k.serviceLister.List()
	result := make([]stateEvent.ServiceInfo, len(services))

	i := 0
	for k, v := range services {
		proposal := v.Proposal()
		result[i] = stateEvent.ServiceInfo{
			ID:         string(k),
			ProviderID: proposal.ProviderID,
			Type:       proposal.ServiceType,
			Options:    v.Options(),
			Status:     string(v.State()),
			Proposal:   proposal,
		}
		i++
	}

	k.state.Services = result
}

func (k *Keeper) updateNatStatus(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	event, ok := e.(natEvent.Event)
	if !ok {
		log.Warn("received a non nat event on nat status call - ignoring")
		return
	}

	k.natStatusProvider.ConsumeNATEvent(event)
	status := k.natStatusProvider.Status()
	k.state.NATStatus = stateEvent.NATStatus{Status: status.Status}
	if status.Error != nil {
		k.state.NATStatus.Error = status.Error.Error()
	}

	go k.publisher.Publish(stateEvent.Topic, *k.state)
}

func (k *Keeper) updateSessionState(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	sessions := k.serviceSessionStorage.GetAll()
	result := make([]stateEvent.ServiceSession, len(sessions))
	for i := range sessions {
		result[i] = stateEvent.ServiceSession{
			ID:         string(sessions[i].ID),
			ConsumerID: sessions[i].ConsumerID.Address,
			CreatedAt:  sessions[i].CreatedAt,
			BytesOut:   sessions[i].DataTransfered.Up,
			BytesIn:    sessions[i].DataTransfered.Down,
		}
	}

	k.state.Sessions = result
	go k.publisher.Publish(stateEvent.Topic, *k.state)
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
