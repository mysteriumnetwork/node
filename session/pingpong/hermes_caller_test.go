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
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/stretchr/testify/assert"
)

func TestHermesCaller_RequestPromise_OK(t *testing.T) {
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
	caller := NewHermesCaller(c, server.URL)
	p, err := caller.RequestPromise(RequestPromise{})
	assert.Nil(t, err)

	assert.EqualValues(t, promise, p)
}

func TestHermesCaller_RequestPromise_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewHermesCaller(c, server.URL)
	_, err := caller.RequestPromise(RequestPromise{})
	assert.NotNil(t, err)
}

func TestHermesCaller_RevealR_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewHermesCaller(c, server.URL)
	err := caller.RevealR("r", "provider", 1)
	assert.NotNil(t, err)
}

func TestHermesCaller_RevealR_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{
			"message": "R successfully revealed"
		  }`))
		assert.NoError(t, err)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewHermesCaller(c, server.URL)
	err := caller.RevealR("r", "provider", 1)
	assert.Nil(t, err)
}

func TestHermesGetConsumerData_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewHermesCaller(c, server.URL)
	_, err := caller.GetConsumerData("something")
	assert.NotNil(t, err)
}

func TestHermesCaller_UnmarshalsErrors(t *testing.T) {
	for k, v := range hermesCauseToError {
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
		caller := NewHermesCaller(c, server.URL)
		err := caller.RevealR("r", "provider", 1)
		assert.EqualError(t, errors.Unwrap(err), v.Error())
		server.Close()
	}
}

func TestHermesGetConsumerData_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		bytes := []byte(mockConsumerData)
		w.Write(bytes)
	}))
	defer server.Close()

	c := requests.NewHTTPClient("0.0.0.0", time.Second)
	caller := NewHermesCaller(c, server.URL)
	data, err := caller.GetConsumerData("0x75C2067Ca5B42467FD6CD789d785aafb52a6B95b")
	assert.Nil(t, err)
	res, err := json.Marshal(data)
	assert.Nil(t, err)

	assert.JSONEq(t, mockConsumerData, string(res))
}

var mockConsumerData = `
{
	"Identity": "0x75C2067Ca5B42467FD6CD789d785aafb52a6B95b",
	"Beneficiary": "0x0000000000000000000000000000000000000000",
	"ChannelID": "0x6295502615e5dDfd1FC7bD22EA5b78d65751A835",
	"Balance": 12185543791,
	"Promised": 217345248,
	"Settled": 0,
	"Stake": 0,
	"LatestPromise": {
	"ChannelID": "0x6295502615e5ddfd1fc7bd22ea5b78d65751a835",
	"Amount": 461730032,
	"Fee": 0,
	"R": null,
	"Hashlock": "0x31c88b635e72755012289cd04bf9b34a11a95f5962f8f1b15dc4b6b80d4af34a",
	"Signature": "0x28d4f2a8c1e2a6b8943e3e110b1d5f66cacaee0841dd7e60ed89e02096419b27188b7c74a9fa1e30e29b4fd75877f503c5d2b193d1d64d7d56232a67b0a102261b"
	},
	"LatestSettlement": "0001-01-01T00:00:00Z"
	}
`

func TestLatestPromise_isValid(t *testing.T) {
	type fields struct {
		ChannelID string
		Amount    uint64
		Fee       uint64
		Hashlock  string
		R         interface{}
		Signature string
	}
	tests := []struct {
		name    string
		fields  fields
		id      string
		wantErr bool
	}{
		{
			name:    "returns no error for a valid promise",
			wantErr: false,
			id:      "0x75C2067Ca5B42467FD6CD789d785aafb52a6B95b",
			fields: fields{
				ChannelID: "0x6295502615e5ddfd1fc7bd22ea5b78d65751a835",
				Amount:    461730032,
				Fee:       0,
				Hashlock:  "0x31c88b635e72755012289cd04bf9b34a11a95f5962f8f1b15dc4b6b80d4af34a",
				Signature: "0x28d4f2a8c1e2a6b8943e3e110b1d5f66cacaee0841dd7e60ed89e02096419b27188b7c74a9fa1e30e29b4fd75877f503c5d2b193d1d64d7d56232a67b0a102261b",
			},
		},
		{
			name:    "returns no error for a valid promise with no prefix on identity",
			wantErr: false,
			id:      "75C2067Ca5B42467FD6CD789d785aafb52a6B95b",
			fields: fields{
				ChannelID: "0x6295502615e5ddfd1fc7bd22ea5b78d65751a835",
				Amount:    461730032,
				Fee:       0,
				Hashlock:  "0x31c88b635e72755012289cd04bf9b34a11a95f5962f8f1b15dc4b6b80d4af34a",
				Signature: "0x28d4f2a8c1e2a6b8943e3e110b1d5f66cacaee0841dd7e60ed89e02096419b27188b7c74a9fa1e30e29b4fd75877f503c5d2b193d1d64d7d56232a67b0a102261b",
			},
		},
		{
			name:    "returns error for a invalid promise",
			wantErr: true,
			id:      "0x75C2067Ca5B42467FD6CD789d785aafb52a6B95b",
			fields: fields{
				ChannelID: "0x3295502615e5ddfd1fc7bd22ea5b78d65751a835",
				Amount:    461730032,
				Fee:       0,
				Hashlock:  "0x31c88b635e72755012289cd04bf9b34a11a95f5962f8f1b15dc4b6b80d4af34a",
				Signature: "0x28d4f2a8c1e2a6b8943e3e110b1d5f66cacaee0841dd7e60ed89e02096419b27188b7c74a9fa1e30e29b4fd75877f503c5d2b193d1d64d7d56232a67b0a102261b",
			},
		},
		{
			name:    "returns error for a invalid hex value",
			wantErr: true,
			id:      "0x75C2067Ca5B42467FD6CD789d785aafb52a6B95b",
			fields: fields{
				ChannelID: "0x3295502615e5ddfd1fc7bd22ea5b78d65751a835",
				Amount:    461730032,
				Fee:       0,
				Hashlock:  "0x0x31c88b635e72755012289cd04bf9b34a11a95f5962f8f1b15dc4b6b80d4af34a",
				Signature: "0x28d4f2a8c1e2a6b8943e3e110b1d5f66cacaee0841dd7e60ed89e02096419b27188b7c74a9fa1e30e29b4fd75877f503c5d2b193d1d64d7d56232a67b0a102261b",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := LatestPromise{
				ChannelID: tt.fields.ChannelID,
				Amount:    tt.fields.Amount,
				Fee:       tt.fields.Fee,
				Hashlock:  tt.fields.Hashlock,
				R:         tt.fields.R,
				Signature: tt.fields.Signature,
			}
			err := lp.isValid(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("LatestPromise.isValid() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
