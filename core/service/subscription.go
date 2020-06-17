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

package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/rs/zerolog/log"
)

func subscribeSessionCreate(mng *session.Manager, ch p2p.Channel, service Service, eventPublisher eventbus.Publisher) {
	ch.Handle(p2p.TopicSessionCreate, func(c p2p.Context) error {
		var sessionInstance *session.Session

		tracer := trace.NewTracer()
		sessionCreateTrace := tracer.StartStage("Provider whole session create")

		defer func() {
			tracer.EndStage(sessionCreateTrace)
			traceResult := tracer.Finish(eventPublisher, string(sessionInstance.ID))
			log.Debug().Msgf("Provider connection trace: %s", traceResult)

			eventPublisher.Publish(servicestate.AppTopicServiceSession, connection.AppEventConnectionSession{
				Status: connection.SessionEndedStatus,
				SessionInfo: connection.Status{
					SessionID: sessionInstance.ID,
					Proposal:  sessionInstance.Proposal,
				},
			})
		}()

		sessionStartTrace := tracer.StartStage("Provider session start")
		var sr pb.SessionRequest
		if err := c.Request().UnmarshalProto(&sr); err != nil {
			return err
		}
		log.Debug().Msgf("Received P2P message for %q: %s", p2p.TopicSessionCreate, sr.String())

		consumerID := identity.FromAddress(sr.GetConsumer().GetId())
		consumerConfig := sr.GetConfig()
		hermesID := common.HexToAddress(sr.GetConsumer().GetAccountantID())
		sessionInstance, err := mng.Start(consumerID, hermesID, int(sr.GetProposalID()))
		if err != nil {
			return fmt.Errorf("cannot start session %s: %w", string(sessionInstance.ID), err)
		}
		tracer.EndStage(sessionStartTrace)

		eventPublisher.Publish(servicestate.AppTopicServiceSession, connection.AppEventConnectionSession{
			Status: connection.SessionCreatedStatus,
			SessionInfo: connection.Status{
				ConsumerID: hermesID,
				HermesID:   accountantID,
				SessionID:  sessionInstance.ID,
				Proposal:   sessionInstance.Proposal,
			},
		})

		provideConfigTrace := tracer.StartStage("Provider config")
		config, err := service.ProvideConfig(string(sessionInstance.ID), consumerConfig, ch.ServiceConn())
		if err != nil {
			return fmt.Errorf("cannot get provider config for session %s: %w", string(sessionInstance.ID), err)
		}
		tracer.EndStage(provideConfigTrace)

		if config.SessionDestroyCallback != nil {
			go func() {
				<-sessionInstance.Done()
				config.SessionDestroyCallback()
			}()
		}

		data, err := json.Marshal(config.SessionServiceConfig)
		if err != nil {
			return fmt.Errorf("cannot pack session %s service config: %w", string(sessionInstance.ID), err)
		}

		pc := p2p.ProtoMessage(&pb.SessionResponse{
			ID:          string(sessionInstance.ID),
			PaymentInfo: string(session.PaymentVersionV3),
			Config:      data,
		})
		return c.OkWithReply(pc)
	})
}

func subscribeSessionStatus(mng *session.Manager, ch p2p.ChannelHandler, statusStorage connectivity.StatusStorage) {
	ch.Handle(p2p.TopicSessionStatus, func(c p2p.Context) error {
		var ss pb.SessionStatus
		if err := c.Request().UnmarshalProto(&ss); err != nil {
			return err
		}
		log.Debug().Msgf("Received P2P session status message for %q: %s", p2p.TopicSessionStatus, ss.String())

		entry := connectivity.StatusEntry{
			PeerID:       identity.FromAddress(ss.GetConsumerID()),
			StatusCode:   connectivity.StatusCode(ss.GetCode()),
			SessionID:    ss.GetSessionID(),
			Message:      ss.GetMessage(),
			CreatedAtUTC: time.Now().UTC(),
		}
		statusStorage.AddStatusEntry(entry)

		return c.OK()
	})
}

func subscribeSessionDestroy(mng *session.Manager, ch p2p.ChannelHandler, done func()) {
	ch.Handle(p2p.TopicSessionDestroy, func(c p2p.Context) error {
		var si pb.SessionInfo
		if err := c.Request().UnmarshalProto(&si); err != nil {
			return err
		}
		log.Debug().Msgf("Received P2P message for %q: %s", p2p.TopicSessionDestroy, si.String())

		go func() {
			consumerID := identity.FromAddress(si.GetConsumerID())
			sessionID := si.GetSessionID()

			err := mng.Destroy(consumerID, sessionID)
			if err != nil {
				log.Err(err).Msgf("Could not destroy session %s: %v", sessionID, err)
			}

			done()
		}()

		return c.OK()
	})
}

func subscribeSessionAcknowledge(mng *session.Manager, ch p2p.ChannelHandler) {
	ch.Handle(p2p.TopicSessionAcknowledge, func(c p2p.Context) error {
		var si pb.SessionInfo
		if err := c.Request().UnmarshalProto(&si); err != nil {
			return err
		}
		log.Debug().Msgf("Received P2P message for %q: %s", p2p.TopicSessionAcknowledge, si.String())
		consumerID := identity.FromAddress(si.GetConsumerID())
		sessionID := si.GetSessionID()

		err := mng.Acknowledge(consumerID, sessionID)
		if err != nil {
			return fmt.Errorf("cannot acknowledge session %s: %w", sessionID, err)
		}

		return c.OK()
	})
}
