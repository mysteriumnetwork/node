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

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/core/state/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/nat"
	natEvent "github.com/mysteriumnetwork/node/nat/event"
	"github.com/mysteriumnetwork/node/session"
	sevent "github.com/mysteriumnetwork/node/session/event"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
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

type identityProvider interface {
	GetIdentities() []identity.Identity
}

type channelAddressCalculator interface {
	GetChannelAddress(id identity.Identity) (common.Address, error)
}

type balanceProvider interface {
	GetBalance(id identity.Identity) uint64
}

type earningsProvider interface {
	GetEarnings(id identity.Identity) pingpongEvent.Earnings
}

// Keeper keeps track of state through eventual consistency.
// This should become the de-facto place to get your info about node.
type Keeper struct {
	state                  *stateEvent.State
	lock                   sync.RWMutex
	sessionConnectionCount map[string]event.ConnectionStatistics
	deps                   KeeperDeps

	// provider
	consumeServiceStateEvent                 func(e interface{})
	consumeNATEvent                          func(e interface{})
	consumeServiceSessionStateEventDebounced func(e interface{})
	// consumer
	consumeConnectionStatisticsEvent func(interface{})
	consumeConnectionThroughputEvent func(interface{})
	consumeConnectionSpendingEvent   func(interface{})

	announceStateChanges func(e interface{})
}

// KeeperDeps to construct the state.Keeper.
type KeeperDeps struct {
	NATStatusProvider         natStatusProvider
	Publisher                 publisher
	ServiceLister             serviceLister
	ServiceSessionStorage     serviceSessionStorage
	IdentityProvider          identityProvider
	IdentityRegistry          registry.IdentityRegistry
	IdentityChannelCalculator channelAddressCalculator
	BalanceProvider           balanceProvider
	EarningsProvider          earningsProvider
}

// NewKeeper returns a new instance of the keeper.
func NewKeeper(deps KeeperDeps, debounceDuration time.Duration) *Keeper {
	k := &Keeper{
		state: &stateEvent.State{
			NATStatus: stateEvent.NATStatus{
				Status: "not_finished",
			},
			Connection: stateEvent.Connection{
				Session: connection.Status{
					State: connection.NotConnected,
				},
			},
		},
		deps:                   deps,
		sessionConnectionCount: make(map[string]event.ConnectionStatistics),
	}
	k.state.Identities = k.fetchIdentities()

	// provider
	k.consumeServiceStateEvent = debounce(k.updateServiceState, debounceDuration)
	k.consumeNATEvent = debounce(k.updateNatStatus, debounceDuration)
	k.consumeServiceSessionStateEventDebounced = debounce(k.updateSessionState, debounceDuration)

	// consumer
	k.consumeConnectionStatisticsEvent = debounce(k.updateConnectionStats, debounceDuration)
	k.consumeConnectionThroughputEvent = debounce(k.updateConnectionThroughput, debounceDuration)
	k.consumeConnectionSpendingEvent = debounce(k.updateConnectionSpending, debounceDuration)
	k.announceStateChanges = debounce(k.announceState, debounceDuration)

	return k
}

func (k *Keeper) fetchIdentities() []stateEvent.Identity {
	ids := k.deps.IdentityProvider.GetIdentities()
	identities := make([]event.Identity, len(ids))
	for idx, id := range ids {
		status, err := k.deps.IdentityRegistry.GetRegistrationStatus(id)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not get registration status for %s", id.Address)
			status = registry.Unregistered
		}

		channelAddress, err := k.deps.IdentityChannelCalculator.GetChannelAddress(id)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not calculate channel address for %s", id.Address)
		}

		earnings := k.deps.EarningsProvider.GetEarnings(id)
		stateIdentity := event.Identity{
			Address:            id.Address,
			RegistrationStatus: status,
			ChannelAddress:     channelAddress,
			Balance:            k.deps.BalanceProvider.GetBalance(id),
			Earnings:           earnings.UnsettledBalance,
			EarningsTotal:      earnings.LifetimeBalance,
		}
		identities[idx] = stateIdentity
	}
	return identities
}

// Subscribe subscribes to the event bus.
func (k *Keeper) Subscribe(bus eventbus.Subscriber) error {
	if err := bus.SubscribeAsync(servicestate.AppTopicServiceStatus, k.consumeServiceStateEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(sevent.AppTopicSession, k.consumeServiceSessionStateEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(natEvent.AppTopicTraversal, k.consumeNATEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(connection.AppTopicConnectionState, k.consumeConnectionStateEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(connection.AppTopicConnectionStatistics, k.consumeConnectionStatisticsEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(bandwidth.AppTopicConnectionThroughput, k.consumeConnectionThroughputEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(pingpongEvent.AppTopicInvoicePaid, k.consumeConnectionSpendingEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(identity.AppTopicIdentityCreated, k.consumeIdentityCreatedEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(registry.AppTopicIdentityRegistration, k.consumeIdentityRegistrationEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(pingpongEvent.AppTopicBalanceChanged, k.consumeBalanceChangedEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(pingpongEvent.AppTopicEarningsChanged, k.consumeEarningsChangedEvent); err != nil {
		return err
	}
	return nil
}

func (k *Keeper) announceState(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.deps.Publisher.Publish(stateEvent.AppTopicState, *k.state)
}

func (k *Keeper) updateServiceState(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.updateServices()
	go k.announceStateChanges(nil)
}

// consumeServiceSessionStateEvent consumes the session change events
func (k *Keeper) consumeServiceSessionStateEvent(e sevent.AppEventSession) {
	if e.Status == sevent.AcknowledgedStatus {
		k.consumeServiceSessionAcknowledgeEvent(e)
		return
	}

	k.consumeServiceSessionStateEventDebounced(e)
}

func (k *Keeper) consumeServiceSessionAcknowledgeEvent(e sevent.AppEventSession) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if e.Status != sevent.AcknowledgedStatus {
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
	services := k.deps.ServiceLister.List()
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

	k.deps.NATStatusProvider.ConsumeNATEvent(event)
	status := k.deps.NATStatusProvider.Status()
	k.state.NATStatus = stateEvent.NATStatus{Status: status.Status}
	if status.Error != nil {
		k.state.NATStatus.Error = status.Error.Error()
	}

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateSessionState(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	sessions := k.deps.ServiceSessionStorage.GetAll()
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

func (k *Keeper) consumeConnectionStateEvent(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(connection.AppEventConnectionState)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for connection state update")
		return
	}

	if evt.State == connection.NotConnected {
		k.state.Connection = stateEvent.Connection{}
	}
	k.state.Connection.Session = evt.SessionInfo
	log.Info().Msgf("Session %s", k.state.Connection.String())

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateConnectionStats(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(connection.AppEventConnectionStatistics)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for connection state update")
		return
	}

	k.state.Connection.Statistics = evt.Stats

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateConnectionThroughput(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(bandwidth.AppEventConnectionThroughput)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for connection state update")
		return
	}

	k.state.Connection.Throughput = evt.Throughput

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateConnectionSpending(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(pingpongEvent.AppEventInvoicePaid)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for connection state update")
		return
	}

	k.state.Connection.Invoice = evt.Invoice
	log.Info().Msgf("Session %s", k.state.Connection.String())

	go k.announceStateChanges(nil)
}

func (k *Keeper) consumeBalanceChangedEvent(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(pingpongEvent.AppEventBalanceChanged)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for balance change")
		return
	}
	var id *stateEvent.Identity
	for i := range k.state.Identities {
		if k.state.Identities[i].Address == evt.Identity.Address {
			id = &k.state.Identities[i]
			break
		}
	}
	if id == nil {
		log.Warn().Msgf("Couldn't find a matching identity for balance change: %s", evt.Identity.Address)
		return
	}
	id.Balance = evt.Current
	go k.announceStateChanges(nil)
}

func (k *Keeper) consumeEarningsChangedEvent(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(pingpongEvent.AppEventEarningsChanged)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for earnings change")
		return
	}
	var id *stateEvent.Identity
	for i := range k.state.Identities {
		if k.state.Identities[i].Address == evt.Identity.Address {
			id = &k.state.Identities[i]
			break
		}
	}
	if id == nil {
		log.Warn().Msgf("Couldn't find a matching identity for earnings change: %s", evt.Identity.Address)
		return
	}
	id.Earnings = evt.Current.UnsettledBalance
	id.EarningsTotal = evt.Current.LifetimeBalance
	go k.announceStateChanges(nil)
}

func (k *Keeper) consumeIdentityCreatedEvent(_ interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	k.state.Identities = k.fetchIdentities()
	go k.announceStateChanges(nil)
}

func (k *Keeper) consumeIdentityRegistrationEvent(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(registry.AppEventIdentityRegistration)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for identity registration")
	}
	var id *stateEvent.Identity
	for i := range k.state.Identities {
		if k.state.Identities[i].Address == evt.ID.Address {
			id = &k.state.Identities[i]
			break
		}
	}
	if id == nil {
		log.Warn().Msgf("Couldn't find a matching identity for balance change: %s", evt.ID.Address)
		return
	}
	id.RegistrationStatus = evt.Status
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
