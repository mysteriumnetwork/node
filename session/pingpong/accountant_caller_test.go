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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestAccountantCaller_RequestPromise_OK(t *testing.T) {
	promise := crypto.Promise{
		ChannelID: []byte("ChannelID"),
		Amount:    1,
		Fee:       1,
		Hashlock:  []byte("lock"),
		R:         []byte("R"),
		Signature: []byte("Signature"),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		bytes, err := json.Marshal(promise)
		assert.Nil(t, err)
		w.Write(bytes)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewAccountantCaller(c, server.URL)
	p, err := caller.RequestPromise(crypto.ExchangeMessage{})
	assert.Nil(t, err)

	assert.EqualValues(t, promise, p)
}

func TestAccountantCaller_RequestPromise_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewAccountantCaller(c, server.URL)
	_, err := caller.RequestPromise(crypto.ExchangeMessage{})
	assert.NotNil(t, err)
}

func TestAccountantCaller_RevealR_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewAccountantCaller(c, server.URL)
	err := caller.RevealR("r", "provider", 1)
	assert.NotNil(t, err)
}

func TestAccountantCaller_RevealR_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"message": "R succesfully revealed"
		  }`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewAccountantCaller(c, server.URL)
	err := caller.RevealR("r", "provider", 1)
	assert.Nil(t, err)
}

func TestAccountantGetConsumerData_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewAccountantCaller(c, server.URL)
	_, err := caller.GetConsumerData("something")
	assert.NotNil(t, err)
}

func TestAccountantCaller_UnmarshalsErrors(t *testing.T) {
	for k, v := range accountantCauseToError {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, err := w.Write([]byte(fmt.Sprintf(`{
				"cause": %q,
				"message": "some message"
			  }`, k)))
			assert.NoError(t, err)
		}))
		defer server.Close()

		c := requests.NewHTTPClient("0.0.0.0", time.Second)
		caller := NewAccountantCaller(c, server.URL)
		err := caller.RevealR("r", "provider", 1)
		assert.Equal(t, v, errors.Cause(err))
		server.Close()
	}
}

func TestAccountantGetConsumerData_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		bytes := []byte(mockConsumerData)
		w.Write(bytes)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewAccountantCaller(c, server.URL)
	data, err := caller.GetConsumerData("something")
	assert.Nil(t, err)
	res, err := json.Marshal(data)
	assert.Nil(t, err)

	assert.JSONEq(t, mockConsumerData, string(res))
}

var mockConsumerData = `
{
	"Identity": "0x7a50ba299c6da1d82799f2a6221174b9c72e824f",
	"Beneficiary": "0x0000000000000000000000000000000000000000",
	"ChannelID": "MHhCZDk5ZEQyOTYxOUZCNDMyMjYxYTdjOUY4ODhFODU3OTNhODBFYjlC",
	"Balance": 12400000000,
	"Promised": 851866,
	"Settled": 0,
	"Stake": 0,
	"LatestPromise": {
	  "ChannelID": "vZndKWGftDImGnyfiI6FeTqA65s=",
	  "Amount": 851866,
	  "Fee": 100000000,
	  "Hashlock": "H3y0u4B5kKSFgu1abk8NRLQYrd2x9/EFBOhFgRSQoeo=",
	  "R": null,
	  "Signature": "S4fonNmmxLh1bblPfs98I2iP/5UGYWwb7rxpnwkS0d41oOuXOGxvzLZWduwOinrS97t/ToRaY8vbq/0MfZ2qARs="
	},
	"LatestSettlement": "0001-01-01T00:00:00Z"
}
`
