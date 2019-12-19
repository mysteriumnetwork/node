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
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
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
		peerConnectionFactory: func(peerContact nats_discovery.ContactNATSV1) (nats.Connection, error) {
			connection, err := nats.NewConnection(peerContact.BrokerAddresses...)
			if err != nil {
				return nil, err
			}

			return connection, connection.Open()
		},
	}
}

type dialogEstablisher struct {
	ID                    identity.Identity
	Signer                identity.Signer
	peerConnectionFactory func(nats_discovery.ContactNATSV1) (nats.Connection, error)
}

func (e *dialogEstablisher) EstablishDialog(
	peerID identity.Identity,
	peerContact market.Contact,
) (communication.Dialog, error) {
	log.Info().Msgf("Establishing dialog to: %#v", peerContact)
	peerContactNats, err := validateContact(peerContact)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid contact: %#v", peerContact)
	}

	peerConnection, err := e.peerConnectionFactory(peerContactNats)
	if err := peerConnection.Open(); err != nil {
		return nil, errors.Wrapf(err, "failed to connect to: %#v", peerContact)
	}
	peerCodec := e.newCodecForPeer(peerID)

	topic, err := e.negotiateMyTopic(peerConnection, peerCodec, peerContactNats.Topic)
	if err != nil {
		peerConnection.Close()
		return nil, err
	}

	dialog := e.newDialogToPeer(peerID, peerConnection, peerCodec, topic)
	log.Info().Msgf("Dialog established with: %#v", peerContact)

	return dialog, nil
}

func validateContact(contact market.Contact) (nats_discovery.ContactNATSV1, error) {
	if contact.Type != nats_discovery.TypeContactNATSV1 {
		return nats_discovery.ContactNATSV1{}, errors.Errorf("invalid contact type: %s", contact.Type)
	}

	contactNats, ok := contact.Definition.(nats_discovery.ContactNATSV1)
	if !ok {
		return nats_discovery.ContactNATSV1{}, errors.Errorf("invalid contact definition: %#v", contact.Definition)
	}

	return contactNats, nil
}

func (e *dialogEstablisher) negotiateMyTopic(
	peerConnection nats.Connection,
	peerCodec *codecSecured,
	peerTopic string,
) (string, error) {
	sender := nats.NewSender(peerConnection, peerCodec, peerTopic)
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

	myTopic := response.(*dialogCreateResponse).Topic
	if len(myTopic) == 0 {
		// TODO this is a compatibility check. It should be removed once all consumers will migrate to the newer version.
		myTopic = peerTopic + "." + e.ID.Address
	}
	return myTopic, nil
}

func (e *dialogEstablisher) newCodecForPeer(peerID identity.Identity) *codecSecured {
	return NewCodecSecured(
		communication.NewCodecJSON(),
		e.Signer,
		identity.NewVerifierIdentity(peerID),
	)
}

func (e *dialogEstablisher) newDialogToPeer(
	peerID identity.Identity,
	peerConnection nats.Connection,
	peerCodec *codecSecured,
	topic string,
) *dialog {
	return &dialog{
		peerID:         peerID,
		peerConnection: peerConnection,
		Sender:         nats.NewSender(peerConnection, peerCodec, topic),
		Receiver:       nats.NewReceiver(peerConnection, peerCodec, topic),
	}
}
