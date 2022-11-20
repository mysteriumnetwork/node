/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package endpoints

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/consumer/session"
	nodeEvent "github.com/mysteriumnetwork/node/core/node/event"
	"github.com/mysteriumnetwork/node/core/state/event"
	stateEvent "github.com/mysteriumnetwork/node/core/state/event"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

// EventType represents all the event types we're subscribing to
type EventType string

// Event represents an event we're gonna send
type Event struct {
	Payload interface{} `json:"payload"`
	Type    EventType   `json:"type"`
}

const (
	// NATEvent represents the nat event type
	NATEvent EventType = "nat"
	// ServiceStatusEvent represents the service status event type
	ServiceStatusEvent EventType = "service-status"
	// StateChangeEvent represents the state change
	StateChangeEvent EventType = "state-change"
)

// Handler represents an sse handler
type Handler struct {
	clients       map[chan string]struct{}
	newClients    chan chan string
	deadClients   chan chan string
	messages      chan string
	stopOnce      sync.Once
	stopChan      chan struct{}
	stateProvider stateProvider
}

type stateProvider interface {
	GetState() stateEvent.State
	GetConnection(string) stateEvent.Connection
}

// NewSSEHandler returns a new instance of handler
func NewSSEHandler(stateProvider stateProvider) *Handler {
	return &Handler{
		clients:       make(map[chan string]struct{}),
		newClients:    make(chan (chan string)),
		deadClients:   make(chan (chan string)),
		messages:      make(chan string, 20),
		stopChan:      make(chan struct{}),
		stateProvider: stateProvider,
	}
}

// Subscribe subscribes to the event bus.
func (h *Handler) Subscribe(bus eventbus.Subscriber) error {
	err := bus.Subscribe(nodeEvent.AppTopicNode, h.ConsumeNodeEvent)
	if err != nil {
		return err
	}
	err = bus.Subscribe(stateEvent.AppTopicState, h.ConsumeStateEvent)
	return err
}

// Sub subscribes a user to sse
func (h *Handler) Sub(c *gin.Context) {
	resp := c.Writer
	req := c.Request

	f, ok := resp.(http.Flusher)
	if !ok {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Header().Set("Content-type", "application/json; charset=utf-8")
		writeErr := json.NewEncoder(resp).Encode(errors.New("not a flusher - cannot continue"))
		if writeErr != nil {
			http.Error(resp, "Http response write error", http.StatusInternalServerError)
		}
		return
	}

	resp.Header().Set("Content-Type", "text/event-stream")
	resp.Header().Set("Cache-Control", "no-cache,no-transform")
	resp.Header().Set("Connection", "keep-alive")

	messageChan := make(chan string, 1)
	err := h.sendInitialState(messageChan)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		resp.Header().Set("Content-type", "application/json; charset=utf-8")
		writeErr := json.NewEncoder(resp).Encode(err)
		if writeErr != nil {
			http.Error(resp, "Http response write error", http.StatusInternalServerError)
		}
	}

	h.newClients <- messageChan

	defer func() {
		h.deadClients <- messageChan
	}()

	for {
		select {
		case <-req.Context().Done():
			return
		case msg, open := <-messageChan:
			if !open {
				return
			}

			_, err := fmt.Fprintf(resp, "data: %s\n\n", msg)
			if err != nil {
				log.Error().Err(err).Msg("failed to print data in response")
				return
			}

			f.Flush()
		case <-h.stopChan:
			return
		}
	}
}

func (h *Handler) sendInitialState(messageChan chan string) error {
	res, err := json.Marshal(Event{
		Type:    StateChangeEvent,
		Payload: mapState(h.stateProvider.GetState()),
	})
	if err != nil {
		return err
	}

	messageChan <- string(res)
	return nil
}

func (h *Handler) serve() {
	defer func() {
		for k := range h.clients {
			close(k)
		}
	}()

	for {
		select {
		case <-h.stopChan:
			return
		case s := <-h.newClients:
			h.clients[s] = struct{}{}
		case s := <-h.deadClients:
			delete(h.clients, s)
			close(s)
		case msg := <-h.messages:
			for s := range h.clients {
				// non-locking send to each client
				select {
				case s <- msg:
				default:
				}
			}
		}
	}
}

func (h *Handler) stop() {
	h.stopOnce.Do(func() { close(h.stopChan) })
}

func (h *Handler) send(e Event) {
	marshaled, err := json.Marshal(e)
	if err != nil {
		log.Error().Err(err).Msg("Could not marshal SSE message")
		return
	}
	h.messages <- string(marshaled)
}

// ConsumeNodeEvent consumes the node state event
func (h *Handler) ConsumeNodeEvent(e nodeEvent.Payload) {
	if e.Status == nodeEvent.StatusStarted {
		go h.serve()
		return
	}
	if e.Status == nodeEvent.StatusStopped {
		h.stop()
		return
	}
}

type stateRes struct {
	Services      []contract.ServiceInfoDTO    `json:"service_info"`
	Sessions      []contract.SessionDTO        `json:"sessions"`
	SessionsStats contract.SessionStatsDTO     `json:"sessions_stats"`
	Consumer      consumerStateRes             `json:"consumer"`
	Identities    []contract.IdentityDTO       `json:"identities"`
	Channels      []contract.PaymentChannelDTO `json:"channels"`
}

type consumerStateRes struct {
	Connection contract.ConnectionDTO `json:"connection"`
}

func mapState(state stateEvent.State) stateRes {
	identitiesRes := make([]contract.IdentityDTO, len(state.Identities))
	for idx, identity := range state.Identities {
		stake := new(big.Int)

		if channel := identityChannel(identity.Address, state.ProviderChannels); channel != nil {
			stake = channel.Channel.Stake
		}

		identitiesRes[idx] = contract.IdentityDTO{
			Address:             identity.Address,
			RegistrationStatus:  identity.RegistrationStatus.String(),
			ChannelAddress:      identity.ChannelAddress.Hex(),
			Balance:             identity.Balance,
			BalanceTokens:       contract.NewTokens(identity.Balance),
			Earnings:            identity.Earnings,
			EarningsTokens:      contract.NewTokens(identity.Earnings),
			EarningsTotal:       identity.EarningsTotal,
			EarningsTotalTokens: contract.NewTokens(identity.EarningsTotal),
			Stake:               stake,
			HermesID:            identity.HermesID.Hex(),
			EarningsPerHermes:   contract.NewEarningsPerHermesDTO(identity.EarningsPerHermes),
		}
	}

	channelsRes := make([]contract.PaymentChannelDTO, len(state.ProviderChannels))
	for idx, channel := range state.ProviderChannels {
		channelsRes[idx] = contract.NewPaymentChannelDTO(channel)
	}

	sessionsRes := make([]contract.SessionDTO, len(state.Sessions))
	sessionsStats := session.NewStats()
	for idx, se := range state.Sessions {
		sessionsRes[idx] = contract.NewSessionDTO(se)
		sessionsStats.Add(se)
	}

	conn := event.Connection{Session: connectionstate.Status{State: connectionstate.NotConnected}}

	for k, c := range state.Connections {
		if c.Session.State == "" {
			c.Session.State = connectionstate.NotConnected
		}
		conn = c
		if len(k) > 0 {
			break
		}
	}

	res := stateRes{
		Services:      state.Services,
		Sessions:      sessionsRes,
		SessionsStats: contract.NewSessionStatsDTO(sessionsStats),
		Consumer: consumerStateRes{
			Connection: contract.NewConnectionDTO(conn.Session, conn.Statistics, conn.Throughput, conn.Invoice),
		},
		Identities: identitiesRes,
		Channels:   channelsRes,
	}
	return res
}

func identityChannel(address string, channels []pingpong.HermesChannel) *pingpong.HermesChannel {
	for idx := range channels {
		if channels[idx].Identity.Address == address {
			return &channels[idx]
		}
	}

	return nil
}

// ConsumeStateEvent consumes the state change event
func (h *Handler) ConsumeStateEvent(event stateEvent.State) {
	h.send(Event{
		Type:    StateChangeEvent,
		Payload: mapState(event),
	})
}
