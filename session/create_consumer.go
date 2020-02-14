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

package session

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/communication"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/pkg/errors"
)

// PromiseLoader loads the last known promise info for the given consumer
type PromiseLoader interface {
	LoadPaymentInfo(consumerID, receiverID, issuerID identity.Identity) *promise.PaymentInfo
}

// createConsumer processes session create requests from communication channel.
type createConsumer struct {
	sessionStarter         Starter
	receiverID             identity.Identity
	peerID                 identity.Identity
	providerConfigProvider ConfigProvider
	promiseLoader          PromiseLoader
}

// Starter starts the session.
type Starter interface {
	Start(session *Session, consumerID identity.Identity, consumerInfo ConsumerInfo, proposalID int, config ServiceConfiguration, pingerParams *traversal.Params) error
}

// GetMessageEndpoint returns endpoint where to receive messages
func (consumer *createConsumer) GetRequestEndpoint() communication.RequestEndpoint {
	return endpointSessionCreate
}

// NewRequest creates struct where request from endpoint will be serialized
func (consumer *createConsumer) NewRequest() (requestPtr interface{}) {
	var request CreateRequest
	return &request
}

// Consume handles requests from endpoint and replies with response
func (consumer *createConsumer) Consume(requestPtr interface{}) (response interface{}, err error) {
	request := requestPtr.(*CreateRequest)

	session, err := newSession()
	if err != nil {
		return responseInternalError, errors.Wrap(err, "could not initialize new session")
	}

	// Pass given consumer config to provider's service config provider.
	sessionConfigParams, err := consumer.providerConfigProvider.ProvideConfig(string(session.ID), request.Config)
	if err != nil {
		return responseInternalError, errors.Wrap(err, "could not get provider session config")
	}

	var indicateNewVersion bool
	issuerID := consumer.peerID
	if request.ConsumerInfo != nil {
		issuerID = request.ConsumerInfo.IssuerID
		if request.ConsumerInfo.PaymentVersion == PaymentVersionV3 {
			indicateNewVersion = true
		}
	} else {
		request.ConsumerInfo = &ConsumerInfo{
			IssuerID: issuerID,
		}
	}

	err = consumer.sessionStarter.Start(session, consumer.peerID, *request.ConsumerInfo, request.ProposalID, sessionConfigParams.SessionServiceConfig, sessionConfigParams.TraversalParams)
	if err != nil {
		return createErrorResponse(err), nil
	}

	if sessionConfigParams.SessionDestroyCallback != nil {
		go func() {
			<-session.done
			sessionConfigParams.SessionDestroyCallback()
		}()
	}

	return createResponse(*session, sessionConfigParams.SessionServiceConfig, consumer.promiseLoader.LoadPaymentInfo(consumer.peerID, consumer.receiverID, issuerID), indicateNewVersion), nil
}

func createErrorResponse(err error) CreateResponse {
	switch err {
	case ErrorInvalidProposal:
		return responseInvalidProposal
	default:
		return responseInternalError
	}
}

func createResponse(sessionInstance Session, config ServiceConfiguration, pi *promise.PaymentInfo, indicateNewVersion bool) CreateResponse {
	serializedConfig, err := json.Marshal(config)
	if err != nil {
		// Failed to serialize session
		// TODO Cant expose error to response, some logging should be here
		return responseInternalError
	}

	// let the consumer know we'll support the new payments
	if indicateNewVersion {
		pi.Supports = string(PaymentVersionV3)
	}

	return CreateResponse{
		Success: true,
		Session: SessionDto{
			ID:     sessionInstance.ID,
			Config: serializedConfig,
		},
		PaymentInfo: pi,
	}
}
