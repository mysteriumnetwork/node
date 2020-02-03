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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

// AccountantErrorResponse represents the errors that accountant returns
type AccountantErrorResponse struct {
	Cause   string `json:"cause"`
	Message string `json:"message"`
}

// AccountantCaller represents the http caller for accountant.
type AccountantCaller struct {
	transport         *requests.HTTPClient
	accountantBaseURI string
}

// NewAccountantCaller returns a new instance of accountant caller.
func NewAccountantCaller(transport *requests.HTTPClient, accountantBaseURI string) *AccountantCaller {
	return &AccountantCaller{
		transport:         transport,
		accountantBaseURI: accountantBaseURI,
	}
}

// RequestPromise requests a promise from accountant.
func (ac *AccountantCaller) RequestPromise(em crypto.ExchangeMessage) (crypto.Promise, error) {
	req, err := requests.NewPostRequest(ac.accountantBaseURI, "request_promise", em)
	if err != nil {
		return crypto.Promise{}, errors.Wrap(err, "could not form request_promise request")
	}

	res := crypto.Promise{}
	err = ac.doRequest(req, &res)
	return res, errors.Wrap(err, "could not request promise")
}

// RevealObject represents the reveal request object.
type RevealObject struct {
	R           string
	Provider    string
	AgreementID uint64
}

// RevealR reveals hashlock key 'r' from 'provider' to the accountant for the agreement identified by 'agreementID'.
func (ac *AccountantCaller) RevealR(r, provider string, agreementID uint64) error {
	req, err := requests.NewPostRequest(ac.accountantBaseURI, "reveal_r", RevealObject{
		R:           r,
		Provider:    provider,
		AgreementID: agreementID,
	})
	if err != nil {
		return errors.Wrap(err, "could not form reveal_r request")
	}
	err = ac.doRequest(req, &RevealSuccess{})
	return errors.Wrap(err, "could not reveal R for accountant")
}

// GetConsumerData gets consumer data from accountant
func (ac *AccountantCaller) GetConsumerData(id string) (ConsumerData, error) {
	req, err := requests.NewGetRequest(ac.accountantBaseURI, fmt.Sprintf("data/consumer/%v", id), nil)
	if err != nil {
		return ConsumerData{}, errors.Wrap(err, "could not form consumer data request")
	}
	var resp ConsumerData
	err = ac.doRequest(req, &resp)
	return resp, errors.Wrap(err, "could not request consumer data accountant")
}

func (ac *AccountantCaller) doRequest(req *http.Request, to interface{}) error {
	resp, err := ac.transport.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not execute request")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "could not read response body")
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 300 {
		// parse response
		err = json.Unmarshal(body, &to)
		if err != nil {
			return errors.Wrap(err, "could not unmarshal response body")
		}
		return nil
	}

	// parse error body
	accountantError := AccountantErrorResponse{}
	err = json.Unmarshal(body, &accountantError)
	if err != nil {
		return errors.Wrap(err, "could not unmarshal error body")
	}

	if v, ok := accountantCauseToError[accountantError.Cause]; ok {
		return errors.Wrap(v, accountantError.Message)
	}

	return errors.Wrap(errors.New(accountantError.Cause), "received unknown error")
}

// ConsumerData represents the consumer data
type ConsumerData struct {
	Identity         string        `json:"Identity"`
	Beneficiary      string        `json:"Beneficiary"`
	ChannelID        string        `json:"ChannelID"`
	Balance          uint64        `json:"Balance"`
	Promised         uint64        `json:"Promised"`
	Settled          uint64        `json:"Settled"`
	Stake            uint64        `json:"Stake"`
	LatestPromise    LatestPromise `json:"LatestPromise"`
	LatestSettlement time.Time     `json:"LatestSettlement"`
}

// LatestPromise represents the latest promise
type LatestPromise struct {
	ChannelID string      `json:"ChannelID"`
	Amount    uint64      `json:"Amount"`
	Fee       uint64      `json:"Fee"`
	Hashlock  string      `json:"Hashlock"`
	R         interface{} `json:"R"`
	Signature string      `json:"Signature"`
}

// RevealSuccess represents the reveal success response from accountant
type RevealSuccess struct {
	Message string `json:"message"`
}

// ErrAccountantInvalidSignature indicates that an invalid signature was sent.
var ErrAccountantInvalidSignature = errors.New("invalid signature")

// ErrAccountantInternal represents an internal error.
var ErrAccountantInternal = errors.New("internal error")

// ErrAccountantPreviousRNotRevealed represents that a previous R has not been revealed yet. No actions will be possible before the R is revealed.
var ErrAccountantPreviousRNotRevealed = errors.New("previous R not revealed")

// ErrAccountantPaymentValueTooLow indicates that the agreement total has decreased as opposed to increasing.
var ErrAccountantPaymentValueTooLow = errors.New("payment value too low")

// ErrAccountantProviderBalanceExhausted indicates that the provider has run out of stake and a rebalance is needed.
var ErrAccountantProviderBalanceExhausted = errors.New("provider balance exhausted, please rebalance your channel")

// ErrAccountantPromiseValueTooLow represents an error where the consumer sent a promise with a decreasing total.
var ErrAccountantPromiseValueTooLow = errors.New("promise value too low")

// ErrAccountantOverspend indicates that the consumer has overspent his balance.
var ErrAccountantOverspend = errors.New("consumer does not have enough balance and is overspending")

// ErrAccountantMalformedJSON indicates that the provider has sent an invalid json in the request.
var ErrAccountantMalformedJSON = errors.New("malformed json")

// ErrAccountantNoPreviousPromise indicates that we have no previous knowledge of a promise for the provider.
var ErrAccountantNoPreviousPromise = errors.New("no previous promise found")

// ErrAccountantHashlockMissmatch occurs when an expected hashlock does not match the one sent by provider.
var ErrAccountantHashlockMissmatch = errors.New("hashlock missmatch")

// ErrAccountantNotFound occurs when a requested resource is not found
var ErrAccountantNotFound = errors.New("resource not found")

var accountantCauseToError = map[string]error{
	ErrAccountantInvalidSignature.Error():         ErrAccountantInvalidSignature,
	ErrAccountantInternal.Error():                 ErrAccountantInternal,
	ErrAccountantPreviousRNotRevealed.Error():     ErrAccountantPreviousRNotRevealed,
	ErrAccountantPaymentValueTooLow.Error():       ErrAccountantPaymentValueTooLow,
	ErrAccountantProviderBalanceExhausted.Error(): ErrAccountantProviderBalanceExhausted,
	ErrAccountantPromiseValueTooLow.Error():       ErrAccountantPromiseValueTooLow,
	ErrAccountantOverspend.Error():                ErrAccountantOverspend,
	ErrAccountantMalformedJSON.Error():            ErrAccountantMalformedJSON,
	ErrAccountantNoPreviousPromise.Error():        ErrAccountantNoPreviousPromise,
	ErrAccountantHashlockMissmatch.Error():        ErrAccountantHashlockMissmatch,
	ErrAccountantNotFound.Error():                 ErrAccountantNotFound,
}
