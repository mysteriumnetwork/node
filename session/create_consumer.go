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
)

const consumerLogPrefix = "[session-create-consumer] "

// PromiseLoader loads the last known promise info for the given consumer
type PromiseLoader interface {
	LoadPaymentInfo(consumerID, receiverID, issuerID identity.Identity) *promise.PaymentInfo
}

// createConsumer processes session create requests from communication channel.
type createConsumer struct {
	sessionCreator Creator
	receiverID     identity.Identity
	peerID         identity.Identity
	configProvider ConfigProvider
	promiseLoader  PromiseLoader
}

// Creator defines method for session creation
type Creator interface {
	Create(consumerID identity.Identity, consumerInfo ConsumerInfo, proposalID int, config ServiceConfiguration, pingerPrams *traversal.Params) (Session, error)
}

// GetMessageEndpoint returns endpoint there to receive messages
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

	sessionConfigParams, err := consumer.configProvider(request.Config, &traversal.Params{})
	if err != nil {
		return responseInternalError, err
	}

	var indicateNewVersion bool
	issuerID := consumer.peerID
	if request.ConsumerInfo != nil {
		issuerID = request.ConsumerInfo.IssuerID
		if request.ConsumerInfo.Supports == PaymentVersionV2 {
			indicateNewVersion = true
		}
	} else {
		request.ConsumerInfo = &ConsumerInfo{
			IssuerID: issuerID,
		}
	}

	sessionConfigParams.TraversalParams.RequestConfig = request.Config
	sessionConfigParams.TraversalParams.Cancel = make(chan struct{})

	sessionInstance, err := consumer.sessionCreator.Create(consumer.peerID, *request.ConsumerInfo, request.ProposalID, sessionConfigParams.SessionServiceConfig, sessionConfigParams.TraversalParams)
	switch err {
	case nil:
		if sessionConfigParams.SessionDestroyCallback != nil {
			go func() {
				<-sessionInstance.done
				sessionConfigParams.SessionDestroyCallback()
			}()
		}
		return responseWithSession(sessionInstance, sessionConfigParams.SessionServiceConfig, consumer.promiseLoader.LoadPaymentInfo(consumer.peerID, consumer.receiverID, issuerID), indicateNewVersion), nil
	case ErrorInvalidProposal:
		return responseInvalidProposal, nil
	default:
		return responseInternalError, nil
	}
}

func responseWithSession(sessionInstance Session, config ServiceConfiguration, pi *promise.PaymentInfo, indicateNewVersion bool) CreateResponse {
	serializedConfig, err := json.Marshal(config)
	if err != nil {
		// Failed to serialize session
		// TODO Cant expose error to response, some logging should be here
		return responseInternalError
	}

	// let the consumer know we'll support the new payments
	if indicateNewVersion {
		pi.Supports = string(PaymentVersionV2)
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
