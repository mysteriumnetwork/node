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
			ChainID:   em.Promise.ChainID,
		},
		AgreementID:    em.AgreementID.Text(bigIntBase),
		AgreementTotal: em.AgreementTotal.Text(bigIntBase),
		Provider:       em.Provider,
		Signature:      em.Signature,
		HermesID:       em.HermesID,
		ChainID:        em.ChainID,
	}
	log.Debug().Msgf("Sending P2P message to %q: %s", p2p.TopicPaymentMessage, pMessage.String())

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err := es.ch.Send(ctx, p2p.TopicPaymentMessage, p2p.ProtoMessage(pMessage))
	return err
}
