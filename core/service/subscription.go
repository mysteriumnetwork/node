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

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/session"
)

func subscribeSessionCreate(mng *session.Manager, ch p2p.Channel, service Service, id ID) {
	ch.Handle(p2p.TopicSessionCreate, func(c p2p.Context) error {
		var sr pb.SessionRequest
		if err := c.Request().UnmarshalProto(&sr); err != nil {
			return err
		}

		consumerID := identity.FromAddress(sr.GetConsumer().GetId())
		consumerConfig := sr.GetConfig()
		consumerInfo := session.ConsumerInfo{
			IssuerID:       consumerID,
			AccountantID:   identity.FromAddress(sr.GetConsumer().GetAccountantID()),
			PaymentVersion: session.PaymentVersion(sr.GetConsumer().GetPaymentVersion()),
		}

		paymentVersion := string(session.PaymentVersionV3)
		session := &session.Session{ID: session.ID(id)}
		config, err := service.ProvideConfig(string(session.ID), consumerConfig, ch.ServiceConn())
		if err != nil {
			return fmt.Errorf("cannot get provider config for session %s: %v", string(session.ID), err)
		}

		err = mng.Start(session, consumerID, consumerInfo, int(sr.GetProposalID()), config, nil)
		if err != nil {
			return fmt.Errorf("cannot start session %s: %v", string(session.ID), err)
		}

		if config.SessionDestroyCallback != nil {
			go func() {
				<-session.Done()
				config.SessionDestroyCallback()
				ch.Close()
			}()
		}

		data, err := json.Marshal(config.SessionServiceConfig)
		if err != nil {
			return fmt.Errorf("cannot pack session %s service config: %v", string(session.ID), err)
		}

		pc := p2p.ProtoMessage(&pb.SessionResponse{
			ID:          string(session.ID),
			DNSs:        "", // TODO: Fill this field or check if it's not in data already.
			PaymentInfo: paymentVersion,
			Config:      data,
		})

		return c.OkWithReply(pc)
	})
}

func subscribeSessionDestroy(mng *session.Manager, ch p2p.Channel) {
	ch.Handle(p2p.TopicSessionDestroy, func(c p2p.Context) error {
		var si pb.SessionInfo
		if err := c.Request().UnmarshalProto(&si); err != nil {
			return err
		}

		consumerID := identity.FromAddress(si.GetConsumerID())
		sessionID := si.GetSessionID()

		err := mng.Destroy(consumerID, sessionID)
		if err != nil {
			return fmt.Errorf("cannot destroy session %s: %v", sessionID, err)
		}

		return c.OK()
	})
}

func subscribeSessionAcknowledge(mng *session.Manager, ch p2p.Channel) {
	ch.Handle(p2p.TopicSessionAcknowledge, func(c p2p.Context) error {
		var si pb.SessionInfo
		if err := c.Request().UnmarshalProto(&si); err != nil {
			return err
		}

		consumerID := identity.FromAddress(si.GetConsumerID())
		sessionID := si.GetSessionID()

		err := mng.Acknowledge(consumerID, sessionID)
		if err != nil {
			return fmt.Errorf("cannot acknowledge session %s: %v", sessionID, err)
		}

		return c.OK()
	})
}
