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
	"sync"

	"github.com/gofrs/uuid"
	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/communication/nats"
	nats_discovery "github.com/mysteriumnetwork/node/communication/nats/discovery"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type validator func(peerID identity.Identity) error

// NewDialogWaiter constructs new DialogWaiter which works through NATS connection.
func NewDialogWaiter(
	connection nats.Connection,
	topic string,
	signer identity.Signer,
	validators ...validator,
) *dialogWaiter {
	return &dialogWaiter{
		connection: connection,
		topic:      topic,
		signer:     signer,
		dialogs:    make([]communication.Dialog, 0),
		validators: validators,
	}
}

type dialogWaiter struct {
	connection nats.Connection
	topic      string
	signer     identity.Signer
	dialogs    []communication.Dialog
	validators []validator

	sync.RWMutex
}

// Start registers dialogWaiter with broker (NATS) service
func (waiter *dialogWaiter) Start() (market.Contact, error) {
	contact := market.Contact{
		Type: nats_discovery.TypeContactNATSV1,
		Definition: nats_discovery.ContactNATSV1{
			Topic:           waiter.topic,
			BrokerAddresses: waiter.connection.Servers(),
		},
	}
	log.Info().Msgf("Connecting to: %v", contact)

	if err := waiter.connection.Open(); err != nil {
		return contact, errors.Errorf("failed to start my connection with: %v", contact)
	}

	return contact, nil
}

// Stop disconnects dialogWaiter from broker (NATS) service
func (waiter *dialogWaiter) Stop() error {
	waiter.RLock()
	defer waiter.RUnlock()

	for _, dialog := range waiter.dialogs {
		dialog.Close()
	}
	waiter.connection.Close()
	return nil
}

// ServeDialogs starts accepting dialogs initiated by peers
func (waiter *dialogWaiter) ServeDialogs(dialogHandler communication.DialogHandler) error {
	createDialog := func(request *dialogCreateRequest) (*dialogCreateResponse, error) {
		err := waiter.validateDialogRequest(request)
		if err != nil {
			log.Error().Err(err).Msg("Validation check failed")
			return &responseInvalidIdentity, nil
		}

		uid, err := uuid.NewV4()
		if err != nil {
			log.Error().Err(err).Msg("Failed to generate unique topic")
			return &responseInternalError, errors.Wrap(err, "failed to generate unique topic")
		}

		peerID := identity.FromAddress(request.PeerID)
		peerTopic := uid.String()
		if len(request.Version) == 0 {
			// TODO this is a compatibility check. It should be removed once all consumers will migrate to the newer version.
			peerTopic = waiter.topic + "." + peerID.Address
		}
		dialog := waiter.newDialogToPeer(peerID, waiter.newCodecForPeer(peerID), peerTopic)
		err = dialogHandler.Handle(dialog)
		if err != nil {
			log.Error().Err(err).Msgf("Failed dialog from: %q", request.PeerID)
			return &responseInternalError, nil
		}

		waiter.Lock()
		waiter.dialogs = append(waiter.dialogs, dialog)
		waiter.Unlock()

		log.Info().Msgf("Accepted dialog from: %q", request.PeerID)
		return &dialogCreateResponse{
			Reason:        responseOK.Reason,
			ReasonMessage: responseOK.ReasonMessage,
			Topic:         peerTopic,
		}, nil
	}
	codec := NewCodecSecured(communication.NewCodecJSON(), waiter.signer, identity.NewVerifierSigned())
	receiver := nats.NewReceiver(waiter.connection, codec, waiter.topic)
	return receiver.Respond(&dialogCreateConsumer{Callback: createDialog})
}

func (waiter *dialogWaiter) newCodecForPeer(peerID identity.Identity) *codecSecured {
	return NewCodecSecured(
		communication.NewCodecJSON(),
		waiter.signer,
		identity.NewVerifierIdentity(peerID),
	)
}

func (waiter *dialogWaiter) newDialogToPeer(peerID identity.Identity, peerCodec *codecSecured, topic string) *dialog {
	return &dialog{
		peerID:   peerID,
		Sender:   nats.NewSender(waiter.connection, peerCodec, topic),
		Receiver: nats.NewReceiver(waiter.connection, peerCodec, topic),
	}
}

func (waiter *dialogWaiter) validateDialogRequest(request *dialogCreateRequest) error {
	if request.PeerID == "" {
		return errors.New("no identity provided")
	}

	for _, f := range waiter.validators {
		if err := f(identity.FromAddress(request.PeerID)); err != nil {
			return errors.Wrap(err, "failed to validate dialog request")
		}
	}

	return nil
}
