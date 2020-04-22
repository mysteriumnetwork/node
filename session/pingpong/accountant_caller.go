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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/crypto"
)

// AccountantErrorResponse represents the errors that accountant returns
type AccountantErrorResponse struct {
	CausedBy     string `json:"cause"`
	ErrorMessage string `json:"message"`
	ErrorData    string `json:"data"`
	c            error
}

// Error returns the associated error
func (aer AccountantErrorResponse) Error() string {
	return aer.c.Error()
}

// Cause returns the associated cause
func (aer AccountantErrorResponse) Cause() error {
	return aer.c
}

// Unwrap unwraps the associated error
func (aer AccountantErrorResponse) Unwrap() error {
	return aer.c
}

// Data returns the associated data
func (aer AccountantErrorResponse) Data() string {
	return aer.ErrorData
}

// UnmarshalJSON unmarshals given data to AccountantErrorResponse
func (aer *AccountantErrorResponse) UnmarshalJSON(data []byte) error {
	var s struct {
		CausedBy     string `json:"cause"`
		ErrorMessage string `json:"message"`
		ErrorData    string `json:"data"`
	}

	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("could not unmarshal error data %w", err)
	}

	aer.CausedBy = s.CausedBy
	aer.ErrorMessage = s.ErrorMessage
	aer.ErrorData = s.ErrorData

	if v, ok := accountantCauseToError[s.CausedBy]; ok {
		aer.c = v
		return nil
	}

	return fmt.Errorf("received unknown error: %v", s.CausedBy)
}

type accountantError interface {
	Error() string
	Cause() error
	Data() string
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

// RequestPromise represents the request for a new accountant promise
type RequestPromise struct {
	ExchangeMessage crypto.ExchangeMessage `json:"exchange_message"`
	TransactorFee   uint64                 `json:"transactor_fee"`
	RRecoveryData   string                 `json:"r_recovery_data"`
}

// RequestPromise requests a promise from accountant.
func (ac *AccountantCaller) RequestPromise(rp RequestPromise) (crypto.Promise, error) {
	req, err := requests.NewPostRequest(ac.accountantBaseURI, "request_promise", rp)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not form request_promise request: %w", err)
	}

	eback := backoff.NewConstantBackOff(time.Millisecond * 500)
	boff := backoff.WithMaxRetries(eback, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	boff = backoff.WithContext(boff, ctx)

	res := crypto.Promise{}

	return res, backoff.Retry(func() error {
		err = ac.doRequest(req, &res)
		if err != nil {
			// if too many requests, retry
			if errors.Is(err, ErrTooManyRequests) {
				return err
			}
			// otherwise, do not retry anymore and return the error
			cancel()
			return fmt.Errorf("could not request promise: %w", err)
		}
		return nil
	}, boff)
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
		return fmt.Errorf("could not form reveal_r request: %w", err)
	}

	eback := backoff.NewConstantBackOff(time.Millisecond * 500)
	boff := backoff.WithMaxRetries(eback, 3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	boff = backoff.WithContext(boff, ctx)
	return backoff.Retry(func() error {
		err = ac.doRequest(req, &RevealSuccess{})
		if err != nil {
			// if too many requests, retry
			if errors.Is(err, ErrTooManyRequests) {
				return err
			}
			// otherwise, do not retry anymore and return the error
			cancel()
			return fmt.Errorf("could not reveal R for accountant: %w", err)
		}
		return nil
	}, boff)
}

// GetConsumerData gets consumer data from accountant
func (ac *AccountantCaller) GetConsumerData(id string) (ConsumerData, error) {
	req, err := requests.NewGetRequest(ac.accountantBaseURI, fmt.Sprintf("data/consumer/%v", id), nil)
	if err != nil {
		return ConsumerData{}, fmt.Errorf("could not form consumer data request: %w", err)
	}
	var resp ConsumerData
	err = ac.doRequest(req, &resp)
	if err != nil {
		return ConsumerData{}, fmt.Errorf("could not request consumer data from accountant: %w", err)
	}

	err = resp.LatestPromise.isValid(id)
	if err != nil {
		return ConsumerData{}, fmt.Errorf("could not check promise validity: %w", err)
	}

	return resp, nil
}

func (ac *AccountantCaller) doRequest(req *http.Request, to interface{}) error {
	resp, err := ac.transport.Do(req)
	if err != nil {
		return fmt.Errorf("could not execute request: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not read response body: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode <= 300 {
		// parse response
		err = json.Unmarshal(body, &to)
		if err != nil {
			return fmt.Errorf("could not unmarshal response body: %w", err)
		}
		return nil
	}

	// parse error body
	accountantError := AccountantErrorResponse{}
	err = json.Unmarshal(body, &accountantError)
	if err != nil {
		return fmt.Errorf("could not unmarshal error body: %w", err)
	}

	return accountantError
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

// isValid checks if the promise is really issued by the given identity
func (lp LatestPromise) isValid(id string) error {
	// if we've not promised anything, that's fine for us.
	// handles the case when we've just registered the identity.
	if lp.Amount == 0 {
		return nil
	}

	decodedChannelID, err := hex.DecodeString(strings.TrimPrefix(lp.ChannelID, "0x"))
	if err != nil {
		return fmt.Errorf("could not decode channel ID: %w", err)
	}
	decodedHashlock, err := hex.DecodeString(strings.TrimPrefix(lp.Hashlock, "0x"))
	if err != nil {
		return fmt.Errorf("could not decode hashlock: %w", err)
	}
	decodedSignature, err := hex.DecodeString(strings.TrimPrefix(lp.Signature, "0x"))
	if err != nil {
		return fmt.Errorf("could not decode hashlock: %w", err)
	}

	p := crypto.Promise{
		ChannelID: decodedChannelID,
		Amount:    lp.Amount,
		Fee:       lp.Fee,
		Hashlock:  decodedHashlock,
		Signature: decodedSignature,
	}

	if !p.IsPromiseValid(common.HexToAddress(id)) {
		return fmt.Errorf("promise issued by wrong identity. Expected %q", id)
	}

	return nil
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

// ErrNeedsRRecovery indicates that we need to recover R.
var ErrNeedsRRecovery = errors.New("r recovery required")

// ErrAccountantNoPreviousPromise indicates that we have no previous knowledge of a promise for the provider.
var ErrAccountantNoPreviousPromise = errors.New("no previous promise found")

// ErrAccountantHashlockMissmatch occurs when an expected hashlock does not match the one sent by provider.
var ErrAccountantHashlockMissmatch = errors.New("hashlock missmatch")

// ErrAccountantNotFound occurs when a requested resource is not found
var ErrAccountantNotFound = errors.New("resource not found")

// ErrTooManyRequests occurs when we call the reveal R or request promise errors asynchronously at the same time.
var ErrTooManyRequests = errors.New("too many simultaneous requests")

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
	ErrNeedsRRecovery.Error():                     ErrNeedsRRecovery,
	ErrTooManyRequests.Error():                    ErrTooManyRequests,
}

type rRecoveryDetails struct {
	R           string `json:"r"`
	AgreementID uint64 `json:"agreement_id"`
}
