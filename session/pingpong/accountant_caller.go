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
	"fmt"
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
)

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
	var resp crypto.Promise
	err = ac.transport.DoRequestAndParseResponse(req, &resp)
	return resp, errors.Wrap(err, "could not request promise from accountant")
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
	return errors.Wrap(ac.transport.DoRequest(req), "could not reveal R for accountant")
}

// GetConsumerData gets consumer data from accountant
func (ac *AccountantCaller) GetConsumerData(channel string) (ConsumerData, error) {
	req, err := requests.NewGetRequest(ac.accountantBaseURI, fmt.Sprintf("data/consumer/%v", channel), nil)
	if err != nil {
		return ConsumerData{}, errors.Wrap(err, "could not form consumer data request")
	}
	var resp ConsumerData
	err = ac.transport.DoRequestAndParseResponse(req, &resp)
	return resp, errors.Wrap(err, "could not request consumer data accountant")
}

// ConsumerData represents the consumer data
type ConsumerData struct {
	Identity         string        `json:"Identity"`
	Beneficiary      string        `json:"Beneficiary"`
	ChannelID        string        `json:"ChannelID"`
	Balance          int64         `json:"Balance"`
	Promised         int           `json:"Promised"`
	Settled          int           `json:"Settled"`
	Stake            int           `json:"Stake"`
	LatestPromise    LatestPromise `json:"LatestPromise"`
	LatestSettlement time.Time     `json:"LatestSettlement"`
}

// LatestPromise represents the latest promise
type LatestPromise struct {
	ChannelID string      `json:"ChannelID"`
	Amount    int         `json:"Amount"`
	Fee       int         `json:"Fee"`
	Hashlock  string      `json:"Hashlock"`
	R         interface{} `json:"R"`
	Signature string      `json:"Signature"`
}
