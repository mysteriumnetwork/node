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

package pingpong

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/rs/zerolog/log"
)

const bigIntBase int = 10

// ExchangeRequest structure represents message from service consumer to send a an exchange message.
type ExchangeRequest struct {
	Message crypto.ExchangeMessage `json:"exchangeMessage"`
}

// ExchangeSender is responsible for sending the exchange messages.
type ExchangeSender struct {
	ch p2p.ChannelSender
}

// NewExchangeSender returns a new instance of exchange message sender.
func NewExchangeSender(ch p2p.ChannelSender) *ExchangeSender {
	return &ExchangeSender{
		ch: ch,
	}
}

// Send sends the given exchange message.
func (es *ExchangeSender) Send(em crypto.ExchangeMessage) error {
	pMessage := &pb.ExchangeMessage{
		Promise: &pb.Promise{
			ChannelID: em.Promise.ChannelID,
			Amount:    em.Promise.Amount.Text(bigIntBase),
			Fee:       em.Promise.Fee.Text(bigIntBase),
			Hashlock:  em.Promise.Hashlock,
			R:         em.Promise.R,
			Signature: em.Promise.Signature,
		},
		AgreementID:    em.AgreementID.Text(bigIntBase),
		AgreementTotal: em.AgreementTotal.Text(bigIntBase),
		Provider:       em.Provider,
		Signature:      em.Signature,
		HermesID:       em.HermesID,
	}
	log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicPaymentMessage, pMessage.String())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := es.ch.Send(ctx, p2p.TopicPaymentMessage, p2p.ProtoMessage(pMessage))
	return err
}

func exchangeMessageReceiver(channel p2p.ChannelHandler) (chan crypto.ExchangeMessage, error) {
	exchangeChan := make(chan crypto.ExchangeMessage, 1)

	channel.Handle(p2p.TopicPaymentMessage, func(c p2p.Context) error {
		var msg pb.ExchangeMessage
		if err := c.Request().UnmarshalProto(&msg); err != nil {
			return fmt.Errorf("could not unmarshal exchange message proto: %w", err)
		}
		log.Debug().Msgf("Received P2P message for %q: %s", p2p.TopicPaymentMessage, msg.String())

		amount, ok := new(big.Int).SetString(msg.GetPromise().GetAmount(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field amount of value %v", amount)
		}

		fee, ok := new(big.Int).SetString(msg.GetPromise().GetFee(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field fee of value %v", fee)
		}

		agreementID, ok := new(big.Int).SetString(msg.GetAgreementID(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field agreementID of value %v", agreementID)
		}

		agreementTotal, ok := new(big.Int).SetString(msg.GetAgreementTotal(), bigIntBase)
		if !ok {
			return fmt.Errorf("could not unmarshal field agreementTotal of value %v", agreementTotal)
		}

		exchangeChan <- crypto.ExchangeMessage{
			Promise: crypto.Promise{
				ChannelID: msg.GetPromise().GetChannelID(),
				Amount:    amount,
				Fee:       fee,
				Hashlock:  msg.GetPromise().GetHashlock(),
				R:         msg.GetPromise().GetR(),
				Signature: msg.GetPromise().GetSignature(),
			},
			AgreementID:    agreementID,
			AgreementTotal: agreementTotal,
			Provider:       msg.GetProvider(),
			Signature:      msg.GetSignature(),
			HermesID:       msg.GetHermesID(),
		}

		return nil
	})

	return exchangeChan, nil
}
