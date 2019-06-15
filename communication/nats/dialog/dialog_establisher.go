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
	"fmt"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	"github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
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

const establisherLogPrefix = "[NATS.DialogEstablisher] "

type dialogEstablisher struct {
	ID                 identity.Identity
	Signer             identity.Signer
	peerAddressFactory func(contact market.Contact) (*discovery.AddressNATS, error)
}

func (establisher *dialogEstablisher) EstablishDialog(
	peerID identity.Identity,
	peerContact market.Contact,
) (communication.Dialog, error) {

	log.Info(establisherLogPrefix, fmt.Sprintf("Connecting to: %#v", peerContact))
	peerAddress, err := establisher.peerAddressFactory(peerContact)
	if err != nil {
		return nil, errors.Errorf("failed to connect to: %#v. %s", peerContact, err)
	}

	peerCodec := establisher.newCodecForPeer(peerID)

	peerSender := establisher.newSenderToPeer(peerAddress, peerCodec)
	topic, err := establisher.negotiateTopic(peerSender)
	if err != nil {
		peerAddress.Disconnect()
		return nil, err
	}

	dialog := establisher.newDialogToPeer(peerID, peerAddress, peerCodec, topic)
	log.Info(establisherLogPrefix, fmt.Sprintf("Dialog established with: %#v", peerContact))

	return dialog, nil
}

func (establisher *dialogEstablisher) negotiateTopic(sender communication.Sender) (string, error) {
	response, err := sender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			PeerID:  establisher.ID.Address,
			Version: "v1",
		},
	})
	if err != nil {
		return "", errors.Errorf("dialog creation error. %s", err)
	}
	if response.(*dialogCreateResponse).Reason != 200 {
		return "", errors.Errorf("dialog creation rejected. %#v", response)
	}

	return response.(*dialogCreateResponse).Topic, nil
}

func (establisher *dialogEstablisher) newCodecForPeer(peerID identity.Identity) *codecSecured {

	return NewCodecSecured(
		communication.NewCodecJSON(),
		establisher.Signer,
		identity.NewVerifierIdentity(peerID),
	)
}

func (establisher *dialogEstablisher) newSenderToPeer(
	peerAddress *discovery.AddressNATS,
	peerCodec *codecSecured,
) communication.Sender {

	return nats.NewSender(
		peerAddress.GetConnection(),
		peerCodec,
		peerAddress.GetTopic(),
	)
}

func (establisher *dialogEstablisher) newDialogToPeer(
	peerID identity.Identity,
	peerAddress *discovery.AddressNATS,
	peerCodec *codecSecured,
	topic string,
) *dialog {
	if len(topic) == 0 {
		// TODO this is a compatibility check. It should be removed once all consumers will migrate to the newer version.
		topic = peerAddress.GetTopic() + "." + establisher.ID.Address
	}

	return &dialog{
		peerID:      peerID,
		peerAddress: peerAddress,
		Sender:      nats.NewSender(peerAddress.GetConnection(), peerCodec, topic),
		Receiver:    nats.NewReceiver(peerAddress.GetConnection(), peerCodec, topic),
	}
}
