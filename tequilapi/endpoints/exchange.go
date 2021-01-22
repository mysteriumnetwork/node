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
	"net/http"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/pkg/errors"
)

type exchangeEndpoint struct {
	me mystexchange
}

type mystexchange interface {
	GetMystExchangeRate() (map[string]float64, error)
}

// NewExchangeEndpoint returns a new exchange endpoint
func NewExchangeEndpoint(mystex mystexchange) *exchangeEndpoint {
	return &exchangeEndpoint{
		me: mystex,
	}
}

// swagger:operation GET /exchange/myst/{currency} Exchange ExchangeMyst
// ---
// summary: Returns the myst price in the given currency
// description: Returns the myst price in the given currency (dai is deprecated)
// parameters:
// - name: currency
//   in: path
//   description: Currency to which myst is converted
//   type: string
//   required: true
// responses:
//   200:
//     description: Myst price in given currency
//     schema:
//       "$ref": "#/definitions/CurrencyExchangeDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *exchangeEndpoint) ExchangeMyst(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	var value float64
	currency := strings.ToUpper(params.ByName("currency"))

	rates, err := e.me.GetMystExchangeRate()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	var ok bool
	value, ok = rates[currency]
	if !ok {
		utils.SendError(writer, errors.New("currency not supported"), http.StatusNotFound)
		return
	}

	status := contract.CurrencyExchangeDTO{
		Value:    value,
		Currency: currency,
	}

	utils.WriteAsJSON(status, writer)
}

// AddRoutesForCurrencyExchange attaches exchange endpoints to router.
func AddRoutesForCurrencyExchange(router *httprouter.Router, mystex mystexchange) {
	e := NewExchangeEndpoint(mystex)
	router.GET("/exchange/myst/:currency", e.ExchangeMyst)
}
