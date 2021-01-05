/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package endpoints

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/stretchr/testify/assert"
)

func Test_ExchangeMyst(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	em := &exchangeMock{
		valToReturn: 1.5,
	}

	me := &mechangeMock{
		vals: map[string]float64{
			"BTC": 1.0,
		},
	}

	// Exchange to DAI green path
	resp := httptest.NewRecorder()
	handlerFunc := NewExchangeEndpoint(em, me).ExchangeMyst
	handlerFunc(resp, req, httprouter.Params{{
		Key:   "currency",
		Value: "dai",
	}})

	assert.Equal(t, http.StatusOK, resp.Result().StatusCode)
	parsedResponse := contract.CurrencyExchangeDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)

	assert.Equal(t, em.valToReturn, parsedResponse.Value)
	assert.Equal(t, "DAI", parsedResponse.Currency)

	// Exchange to DAI returns an error
	em.errToReturn = errors.New("boom")
	resp = httptest.NewRecorder()
	handlerFunc(resp, req, httprouter.Params{{
		Key:   "currency",
		Value: "dai",
	}})
	assert.Equal(t, http.StatusInternalServerError, resp.Result().StatusCode)

	// Exchange to BTC green path
	resp = httptest.NewRecorder()
	handlerFunc(resp, req, httprouter.Params{{
		Key:   "currency",
		Value: "btc",
	}})
	assert.Equal(t, http.StatusOK, resp.Result().StatusCode)
	parsedResponse = contract.CurrencyExchangeDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)

	assert.Equal(t, me.vals["BTC"], parsedResponse.Value)
	assert.Equal(t, "BTC", parsedResponse.Currency)

	// No such currency returns 404
	resp = httptest.NewRecorder()
	handlerFunc(resp, req, httprouter.Params{{
		Key:   "currency",
		Value: "notACurrency",
	}})
	assert.Equal(t, http.StatusNotFound, resp.Result().StatusCode)
}

func Test_DaiToMyst(t *testing.T) {
	req, err := http.NewRequest(
		http.MethodGet,
		"/irrelevant",
		nil,
	)
	assert.Nil(t, err)

	em := &exchangeMock{
		valToReturn: 1.5,
	}

	resp := httptest.NewRecorder()
	handlerFunc := NewExchangeEndpoint(em, &mechangeMock{}).DaiToMyst
	handlerFunc(resp, req, nil)

	assert.Equal(t, http.StatusOK, resp.Result().StatusCode)
	parsedResponse := contract.CurrencyExchangeDTO{}
	err = json.Unmarshal(resp.Body.Bytes(), &parsedResponse)
	assert.Nil(t, err)

	assert.Equal(t, em.valToReturn, parsedResponse.Value)
	assert.Equal(t, "MYST", parsedResponse.Currency)

	em.errToReturn = errors.New("boom")
	resp = httptest.NewRecorder()
	handlerFunc(resp, req, nil)
	assert.Equal(t, http.StatusInternalServerError, resp.Result().StatusCode)
}

type mechangeMock struct {
	vals map[string]float64
}

func (m *mechangeMock) GetMystExchangeRate() (map[string]float64, error) {
	return m.vals, nil
}

type exchangeMock struct {
	valToReturn float64
	errToReturn error
}

func (me *exchangeMock) DaiToMyst() (float64, error) {
	return me.valToReturn, me.errToReturn
}

func (me *exchangeMock) MystToDai() (float64, error) {
	return me.valToReturn, me.errToReturn
}
