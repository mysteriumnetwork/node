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
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/consumer/bandwidth"
	"github.com/mysteriumnetwork/node/consumer/session"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/core/state/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/market"
	nodeSession "github.com/mysteriumnetwork/node/session"
	sevent "github.com/mysteriumnetwork/node/session/event"
	"github.com/mysteriumnetwork/node/session/pingpong"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

// DefaultDebounceDuration is the default time interval suggested for debouncing
const DefaultDebounceDuration = time.Millisecond * 200

type publisher interface {
	Publish(topic string, data interface{})
}

type serviceLister interface {
	List() map[service.ID]*service.Instance
}

type identityProvider interface {
	GetIdentities() []identity.Identity
}

type channelAddressCalculator interface {
	GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error)
	GetActiveHermes(chainID int64) (common.Address, error)
}

type balanceProvider interface {
	GetBalance(chainID int64, id identity.Identity) *big.Int
}

type earningsProvider interface {
	List(chainID int64) []pingpong.HermesChannel
	GetEarnings(chainID int64, id identity.Identity) pingpongEvent.Earnings
}

// Keeper keeps track of state through eventual consistency.
// This should become the de-facto place to get your info about node.
type Keeper struct {
	state *stateEvent.State
	lock  sync.RWMutex
	deps  KeeperDeps

	// provider
	consumeServiceStateEvent             func(e interface{})
	consumeServiceSessionStatisticsEvent func(e interface{})
	consumeServiceSessionEarningsEvent   func(e interface{})
	consumeNATStatusUpdateEvent          func(e interface{})
	// consumer
	consumeConnectionStatisticsEvent func(interface{})
	consumeConnectionThroughputEvent func(interface{})
	consumeConnectionSpendingEvent   func(interface{})

	announceStateChanges func(e interface{})
}

// KeeperDeps to construct the state.Keeper.
type KeeperDeps struct {
	Publisher                 publisher
	ServiceLister             serviceLister
	IdentityProvider          identityProvider
	IdentityRegistry          registry.IdentityRegistry
	IdentityChannelCalculator channelAddressCalculator
	BalanceProvider           balanceProvider
	EarningsProvider          earningsProvider
	ChainID                   int64
	ProposalPricer            proposalPricer
}

type proposalPricer interface {
	EnrichProposalWithPrice(in market.ServiceProposal) (proposal.PricedServiceProposal, error)
}

// NewKeeper returns a new instance of the keeper.
func NewKeeper(deps KeeperDeps, debounceDuration time.Duration) *Keeper {
	k := &Keeper{
		state: &stateEvent.State{
			Sessions: make([]session.History, 0),
			Connection: stateEvent.Connection{
				Session: connectionstate.Status{
					State: connectionstate.NotConnected,
				},
			},
		},
		deps: deps,
	}
	k.state.Identities = k.fetchIdentities()
	k.state.ProviderChannels = k.deps.EarningsProvider.List(deps.ChainID)

	// provider
	k.consumeServiceStateEvent = debounce(k.updateServiceState, debounceDuration)
	k.consumeServiceSessionStatisticsEvent = debounce(k.updateSessionStats, debounceDuration)
	k.consumeServiceSessionEarningsEvent = debounce(k.updateSessionEarnings, debounceDuration)

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
		status, err := k.deps.IdentityRegistry.GetRegistrationStatus(k.deps.ChainID, id)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not get registration status for %s", id.Address)
			status = registry.Unknown
		}

		hermesID, err := k.deps.IdentityChannelCalculator.GetActiveHermes(k.deps.ChainID)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not retrieve hermesID for %s", id.Address)
		}

		channelAddress, err := k.deps.IdentityChannelCalculator.GetChannelAddress(k.deps.ChainID, id)
		if err != nil {
			log.Warn().Err(err).Msgf("Could not calculate channel address for %s", id.Address)
		}

		earnings := k.deps.EarningsProvider.GetEarnings(k.deps.ChainID, id)
		stateIdentity := event.Identity{
			Address:            id.Address,
			RegistrationStatus: status,
			ChannelAddress:     channelAddress,
			Balance:            k.deps.BalanceProvider.GetBalance(k.deps.ChainID, id),
			Earnings:           earnings.UnsettledBalance,
			EarningsTotal:      earnings.LifetimeBalance,
			HermesID:           hermesID,
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
	if err := bus.SubscribeAsync(sevent.AppTopicSession, k.consumeServiceSessionEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(sevent.AppTopicDataTransferred, k.consumeServiceSessionStatisticsEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(sevent.AppTopicTokensEarned, k.consumeServiceSessionEarningsEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(connectionstate.AppTopicConnectionState, k.consumeConnectionStateEvent); err != nil {
		return err
	}
	if err := bus.SubscribeAsync(connectionstate.AppTopicConnectionStatistics, k.consumeConnectionStatisticsEvent); err != nil {
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

func (k *Keeper) updateServices() {
	services := k.deps.ServiceLister.List()
	result := make([]contract.ServiceInfoDTO, len(services))

	i := 0
	for key, v := range services {
		// merge in the connection statistics
		match, _ := k.getServiceByID(string(key))

		priced, err := k.deps.ProposalPricer.EnrichProposalWithPrice(v.Proposal)
		if err != nil {
			log.Warn().Msgf("could not load price for proposal %v(%v)", v.Proposal.ProviderID, v.Proposal.ServiceType)
		}

		result[i] = contract.ServiceInfoDTO{
			ID:                   string(key),
			ProviderID:           v.ProviderID.Address,
			Type:                 v.Type,
			Options:              v.Options,
			Status:               string(v.State()),
			Proposal:             contract.NewProposalDTO(priced),
			ConnectionStatistics: match.ConnectionStatistics,
		}
		i++
	}

	k.state.Services = result
}

func (k *Keeper) getServiceByID(id string) (se contract.ServiceInfoDTO, found bool) {
	for i := range k.state.Services {
		if k.state.Services[i].ID == id {
			se = k.state.Services[i]
			found = true
			return
		}
	}
	return
}

// consumeServiceSessionEvent consumes the session change events
func (k *Keeper) consumeServiceSessionEvent(e sevent.AppEventSession) {
	k.lock.Lock()
	defer k.lock.Unlock()

	switch e.Status {
	case sevent.CreatedStatus:
		k.addSession(e)
		k.incrementConnectCount(e.Service.ID, false)
	case sevent.RemovedStatus:
		k.removeSession(e)
	case sevent.AcknowledgedStatus:
		k.incrementConnectCount(e.Service.ID, true)
	}

	go k.announceStateChanges(nil)
}

func (k *Keeper) addSession(e sevent.AppEventSession) {
	k.state.Sessions = append(k.state.Sessions, session.History{
		SessionID:       nodeSession.ID(e.Session.ID),
		Direction:       session.DirectionProvided,
		ConsumerID:      e.Session.ConsumerID,
		HermesID:        e.Session.HermesID.Hex(),
		ProviderID:      identity.FromAddress(e.Session.Proposal.ProviderID),
		ServiceType:     e.Session.Proposal.ServiceType,
		ConsumerCountry: e.Session.ConsumerLocation.Country,
		ProviderCountry: e.Session.Proposal.Location.Country,
		Started:         e.Session.StartedAt,
		Status:          session.StatusNew,
		Tokens:          big.NewInt(0),
	})
}

func (k *Keeper) removeSession(e sevent.AppEventSession) {
	found := false
	for i := range k.state.Sessions {
		if string(k.state.Sessions[i].SessionID) == e.Session.ID {
			k.state.Sessions = append(k.state.Sessions[:i], k.state.Sessions[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		log.Warn().Msgf("Couldn't find a matching session for session remove: %s", e.Session.ID)
	}
}

// updates the data transfer info on the session
func (k *Keeper) updateSessionStats(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	evt, ok := e.(sevent.AppEventDataTransferred)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for session state update")
		return
	}

	var session *session.History
	for i := range k.state.Sessions {
		if string(k.state.Sessions[i].SessionID) == evt.ID {
			session = &k.state.Sessions[i]
		}
	}
	if session == nil {
		log.Warn().Msgf("Couldn't find a matching session for data transferred change: %+v", evt)
		return
	}

	// From a server perspective, bytes up are the actual bytes the client downloaded(aka the bytes we pushed to the consumer)
	// To lessen the confusion, I suggest having the bytes reversed on the session instance.
	// This way, the session will show that it downloaded the bytes in a manner that is easier to comprehend.
	session.DataReceived = evt.Up
	session.DataSent = evt.Down
	go k.announceStateChanges(nil)
}

// updates total tokens earned during the session.
func (k *Keeper) updateSessionEarnings(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()

	evt, ok := e.(sevent.AppEventTokensEarned)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for connection state update")
		return
	}

	var session *session.History
	for i := range k.state.Sessions {
		if string(k.state.Sessions[i].SessionID) == evt.SessionID {
			session = &k.state.Sessions[i]
		}
	}
	if session == nil {
		log.Warn().Msgf("Couldn't find a matching session for earnings change: %s", evt.SessionID)
		return
	}

	session.Tokens = evt.Total
	go k.announceStateChanges(nil)
}

func (k *Keeper) consumeConnectionStateEvent(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(connectionstate.AppEventConnectionState)
	if !ok {
		log.Warn().Msg("Received a wrong kind of event for connection state update")
		return
	}

	if evt.State == connectionstate.NotConnected {
		k.state.Connection = stateEvent.Connection{}
	}
	k.state.Connection.Session = evt.SessionInfo
	log.Info().Msgf("Session %s", k.state.Connection.String())

	go k.announceStateChanges(nil)
}

func (k *Keeper) updateConnectionStats(e interface{}) {
	k.lock.Lock()
	defer k.lock.Unlock()
	evt, ok := e.(connectionstate.AppEventConnectionStatistics)
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

	k.state.ProviderChannels = k.deps.EarningsProvider.List(k.deps.ChainID)

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
