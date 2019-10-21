/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dialog

import (
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// NewDialogEstablisher constructs new DialogEstablisher which works thru NATS connection.
func NewDialogEstablisher(ID identity.Identity, signer identity.Signer) *dialogEstablisher {
	return &dialogEstablisher{
		ID:     ID,
		Signer: signer,
		peerAddressFactory: func(contact market.Contact) (*discovery.AddressNATS, error) {
			address, err := discovery.NewAddressForContact(contact)
			if err == nil {
				err = address.Connect()
			}

			return address, err
		},
	}
}

type dialogEstablisher struct {
	ID                 identity.Identity
	Signer             identity.Signer
	peerAddressFactory func(contact market.Contact) (*discovery.AddressNATS, error)
}

func (e *dialogEstablisher) EstablishDialog(
	peerID identity.Identity,
	peerContact market.Contact,
) (communication.Dialog, error) {

	log.Info().Msgf("Connecting to: %#v", peerContact)
	peerAddress, err := e.peerAddressFactory(peerContact)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to: %#v", peerContact)
	}

	peerCodec := e.newCodecForPeer(peerID)

	peerSender := e.newSenderToPeer(peerAddress, peerCodec)
	topic, err := e.negotiateTopic(peerSender)
	if err != nil {
		peerAddress.Disconnect()
		return nil, err
	}

	dialog := e.newDialogToPeer(peerID, peerAddress, peerCodec, topic)
	log.Info().Msgf("Dialog established with: %#v", peerContact)

	return dialog, nil
}

func (e *dialogEstablisher) negotiateTopic(sender communication.Sender) (string, error) {
	response, err := sender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			PeerID:  e.ID.Address,
			Version: "v1",
		},
	})
	if err != nil {
		return "", errors.Wrapf(err, "dialog creation error")
	}
	if response.(*dialogCreateResponse).Reason != 200 {
		return "", errors.Errorf("dialog creation rejected. %#v", response)
	}

	return response.(*dialogCreateResponse).Topic, nil
}

func (e *dialogEstablisher) newCodecForPeer(peerID identity.Identity) *codecSecured {

	return NewCodecSecured(
		communication.NewCodecJSON(),
		e.Signer,
		identity.NewVerifierIdentity(peerID),
	)
}

func (e *dialogEstablisher) newSenderToPeer(
	peerAddress *discovery.AddressNATS,
	peerCodec *codecSecured,
) communication.Sender {

	return nats.NewSender(
		peerAddress.GetConnection(),
		peerCodec,
		peerAddress.GetTopic(),
	)
}

func (e *dialogEstablisher) newDialogToPeer(
	peerID identity.Identity,
	peerAddress *discovery.AddressNATS,
	peerCodec *codecSecured,
	topic string,
) *dialog {
	if len(topic) == 0 {
		// TODO this is a compatibility check. It should be removed once all consumers will migrate to the newer version.
		topic = peerAddress.GetTopic() + "." + e.ID.Address
	}

	return &dialog{
		peerID:      peerID,
		peerAddress: peerAddress,
		Sender:      nats.NewSender(peerAddress.GetConnection(), peerCodec, topic),
		Receiver:    nats.NewReceiver(peerAddress.GetConnection(), peerCodec, topic),
	}
}
