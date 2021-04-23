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
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/crypto"
)

// HermesErrorResponse represents the errors that hermes returns
type HermesErrorResponse struct {
	CausedBy     string `json:"cause"`
	ErrorMessage string `json:"message"`
	ErrorData    string `json:"data"`
	c            error
}

// Error returns the associated error
func (aer HermesErrorResponse) Error() string {
	return aer.c.Error()
}

// Cause returns the associated cause
func (aer HermesErrorResponse) Cause() error {
	return aer.c
}

// Unwrap unwraps the associated error
func (aer HermesErrorResponse) Unwrap() error {
	return aer.c
}

// Data returns the associated data
func (aer HermesErrorResponse) Data() string {
	return aer.ErrorData
}

// UnmarshalJSON unmarshals given data to HermesErrorResponse
func (aer *HermesErrorResponse) UnmarshalJSON(data []byte) error {
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

	if v, ok := hermesCauseToError[s.CausedBy]; ok {
		aer.c = v
		return nil
	}

	return fmt.Errorf("received unknown error: %v", s.CausedBy)
}

type hermesError interface {
	Error() string
	Cause() error
	Data() string
}

// HermesCaller represents the http caller for hermes.
type HermesCaller struct {
	transport     *requests.HTTPClient
	hermesBaseURI string
}

// NewHermesCaller returns a new instance of hermes caller.
func NewHermesCaller(transport *requests.HTTPClient, hermesBaseURI string) *HermesCaller {
	return &HermesCaller{
		transport:     transport,
		hermesBaseURI: hermesBaseURI,
	}
}

// RequestPromise represents the request for a new hermes promise
type RequestPromise struct {
	ExchangeMessage crypto.ExchangeMessage `json:"exchange_message"`
	TransactorFee   *big.Int               `json:"transactor_fee"`
	RRecoveryData   string                 `json:"r_recovery_data"`
}

// RequestPromise requests a promise from hermes.
func (ac *HermesCaller) RequestPromise(rp RequestPromise) (crypto.Promise, error) {
	return ac.promiseRequest(rp, "request_promise")
}

func (ac *HermesCaller) promiseRequest(rp RequestPromise, endpoint string) (crypto.Promise, error) {
	req, err := requests.NewPostRequest(ac.hermesBaseURI, endpoint, rp)
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not form %v request: %w", endpoint, err)
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

// PayAndSettle requests a promise from hermes.
func (ac *HermesCaller) PayAndSettle(rp RequestPromise) (crypto.Promise, error) {
	return ac.promiseRequest(rp, "pay_and_settle")
}

// SetPromiseFeeRequest represents the payload for changing a promise fee.
type SetPromiseFeeRequest struct {
	HermesPromise crypto.Promise `json:"hermes_promise"`
	NewFee        *big.Int       `json:"new_fee"`
}

// UpdatePromiseFee calls hermes to update its promise with new fee.
func (ac *HermesCaller) UpdatePromiseFee(promise crypto.Promise, newFee *big.Int) (crypto.Promise, error) {
	req, err := requests.NewPostRequest(ac.hermesBaseURI, "change_promise_fee", SetPromiseFeeRequest{
		HermesPromise: promise,
		NewFee:        newFee,
	})
	if err != nil {
		return crypto.Promise{}, fmt.Errorf("could not form change promise fee request: %w", err)
	}

	res := crypto.Promise{}
	return res, ac.doRequest(req, &res)
}

// RevealObject represents the reveal request object.
type RevealObject struct {
	R           string
	Provider    string
	AgreementID *big.Int
}

// RevealR reveals hashlock key 'r' from 'provider' to the hermes for the agreement identified by 'agreementID'.
func (ac *HermesCaller) RevealR(r, provider string, agreementID *big.Int) error {
	req, err := requests.NewPostRequest(ac.hermesBaseURI, "reveal_r", RevealObject{
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
			return fmt.Errorf("could not reveal R for hermes: %w", err)
		}
		return nil
	}, boff)
}

// IsIdentityOffchain returns true if identity is considered offchain in hermes.
func (ac *HermesCaller) IsIdentityOffchain(chainID int64, id string) (bool, error) {
	data, err := ac.GetConsumerData(chainID, id)
	if err != nil {
		if errors.Is(err, ErrHermesNotFound) {
			return false, nil
		}
		return false, err
	}

	return data.IsOffchain, nil
}

// GetConsumerData gets consumer data from hermes
func (ac *HermesCaller) GetConsumerData(chainID int64, id string) (ConsumerData, error) {
	req, err := requests.NewGetRequest(ac.hermesBaseURI, fmt.Sprintf("data/consumer/%v", id), nil)
	if err != nil {
		return ConsumerData{}, fmt.Errorf("could not form consumer data request: %w", err)
	}
	var resp map[int64]ConsumerData
	err = ac.doRequest(req, &resp)
	if err != nil {
		return ConsumerData{}, fmt.Errorf("could not request consumer data from hermes: %w", err)
	}

	data, ok := resp[chainID]
	if !ok {
		return ConsumerData{}, fmt.Errorf("could not get data for chain ID: %d", chainID)
	}

	err = data.LatestPromise.isValid(id)
	if err != nil {
		return ConsumerData{}, fmt.Errorf("could not check promise validity: %w", err)
	}

	return data, nil
}

func (ac *HermesCaller) doRequest(req *http.Request, to interface{}) error {
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
	hermesError := HermesErrorResponse{}
	err = json.Unmarshal(body, &hermesError)
	if err != nil {
		return fmt.Errorf("could not unmarshal error body: %w", err)
	}

	return hermesError
}

// ConsumerData represents the consumer data
type ConsumerData struct {
	Identity         string        `json:"Identity"`
	Beneficiary      string        `json:"Beneficiary"`
	ChannelID        string        `json:"ChannelID"`
	Balance          *big.Int      `json:"Balance"`
	Settled          *big.Int      `json:"Settled"`
	Stake            *big.Int      `json:"Stake"`
	LatestPromise    LatestPromise `json:"LatestPromise"`
	LatestSettlement time.Time     `json:"LatestSettlement"`
	IsOffchain       bool          `json:"IsOffchain"`
}

// LatestPromise represents the latest promise
type LatestPromise struct {
	ChainID   int64    `json:"ChainID"`
	ChannelID string   `json:"ChannelID"`
	Amount    *big.Int `json:"Amount"`
	Fee       *big.Int `json:"Fee"`
	Hashlock  string   `json:"Hashlock"`
	Signature string   `json:"Signature"`
}

// isValid checks if the promise is really issued by the given identity
func (lp LatestPromise) isValid(id string) error {
	// if we've not promised anything, that's fine for us.
	// handles the case when we've just registered the identity.
	if lp.Amount == nil || lp.Amount.Cmp(new(big.Int)) == 0 {
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
		return fmt.Errorf("could not decode signature: %w", err)
	}

	p := crypto.Promise{
		ChainID:   lp.ChainID,
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

// RevealSuccess represents the reveal success response from hermes
type RevealSuccess struct {
	Message string `json:"message"`
}

// ErrHermesInvalidSignature indicates that an invalid signature was sent.
var ErrHermesInvalidSignature = errors.New("invalid signature")

// ErrHermesInternal represents an internal error.
var ErrHermesInternal = errors.New("internal error")

// ErrHermesPreviousRNotRevealed represents that a previous R has not been revealed yet. No actions will be possible before the R is revealed.
var ErrHermesPreviousRNotRevealed = errors.New("previous R not revealed")

// ErrHermesPaymentValueTooLow indicates that the agreement total has decreased as opposed to increasing.
var ErrHermesPaymentValueTooLow = errors.New("payment value too low")

// ErrHermesProviderBalanceExhausted indicates that the provider has run out of stake and a rebalance is needed.
var ErrHermesProviderBalanceExhausted = errors.New("provider balance exhausted, please rebalance your channel")

// ErrHermesPromiseValueTooLow represents an error where the consumer sent a promise with a decreasing total.
var ErrHermesPromiseValueTooLow = errors.New("promise value too low")

// ErrHermesOverspend indicates that the consumer has overspent his balance.
var ErrHermesOverspend = errors.New("consumer does not have enough balance and is overspending")

// ErrHermesMalformedJSON indicates that the provider has sent an invalid json in the request.
var ErrHermesMalformedJSON = errors.New("malformed json")

// ErrNeedsRRecovery indicates that we need to recover R.
var ErrNeedsRRecovery = errors.New("r recovery required")

// ErrHermesNoPreviousPromise indicates that we have no previous knowledge of a promise for the provider.
var ErrHermesNoPreviousPromise = errors.New("no previous promise found")

// ErrHermesHashlockMissmatch occurs when an expected hashlock does not match the one sent by provider.
var ErrHermesHashlockMissmatch = errors.New("hashlock missmatch")

// ErrHermesNotFound occurs when a requested resource is not found
var ErrHermesNotFound = errors.New("resource not found")

// ErrTooManyRequests occurs when we call the reveal R or request promise errors asynchronously at the same time.
var ErrTooManyRequests = errors.New("too many simultaneous requests")

// ErrConsumerUnregistered indicates that the consumer is not registered.
var ErrConsumerUnregistered = errors.New("consumer unregistered")

var hermesCauseToError = map[string]error{
	ErrHermesInvalidSignature.Error():         ErrHermesInvalidSignature,
	ErrHermesInternal.Error():                 ErrHermesInternal,
	ErrHermesPreviousRNotRevealed.Error():     ErrHermesPreviousRNotRevealed,
	ErrHermesPaymentValueTooLow.Error():       ErrHermesPaymentValueTooLow,
	ErrHermesProviderBalanceExhausted.Error(): ErrHermesProviderBalanceExhausted,
	ErrHermesPromiseValueTooLow.Error():       ErrHermesPromiseValueTooLow,
	ErrHermesOverspend.Error():                ErrHermesOverspend,
	ErrHermesMalformedJSON.Error():            ErrHermesMalformedJSON,
	ErrHermesNoPreviousPromise.Error():        ErrHermesNoPreviousPromise,
	ErrHermesHashlockMissmatch.Error():        ErrHermesHashlockMissmatch,
	ErrHermesNotFound.Error():                 ErrHermesNotFound,
	ErrNeedsRRecovery.Error():                 ErrNeedsRRecovery,
	ErrTooManyRequests.Error():                ErrTooManyRequests,
	ErrConsumerUnregistered.Error():           ErrConsumerUnregistered,
}

type rRecoveryDetails struct {
	R           string   `json:"r"`
	AgreementID *big.Int `json:"agreement_id"`
}
