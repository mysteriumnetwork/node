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

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type exchangeEndpoint struct {
	e exchange
}

type exchange interface {
	DaiToMyst() (float64, error)
	MystToDai() (float64, error)
}

// NewExchangeEndpoint returns a new exchange endpoint
func NewExchangeEndpoint(ex exchange) *exchangeEndpoint {
	return &exchangeEndpoint{
		e: ex,
	}
}

// swagger:operation GET /myst/dai myst price in dai
// ---
// summary: Returns the myst price in dai
// description: Returns the myst price in dai
// responses:
//   200:
//     description: Myst price in dai
//     schema:
//       "$ref": "#/definitions/CurrencyExchangeDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *exchangeEndpoint) MystToDai(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	res, err := e.e.MystToDai()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	status := contract.CurrencyExchangeDTO{
		Value:    res,
		Currency: "DAI",
	}

	utils.WriteAsJSON(status, writer)
}

// swagger:operation GET /dai/myst dai price in myst
// ---
// summary: Returns the dai price in myst
// description: Returns the dai price in myst
// responses:
//   200:
//     description: Dai price in myst
//     schema:
//       "$ref": "#/definitions/CurrencyExchangeDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *exchangeEndpoint) DaiToMyst(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	res, err := e.e.DaiToMyst()
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	status := contract.CurrencyExchangeDTO{
		Value:    res,
		Currency: "MYST",
	}

	utils.WriteAsJSON(status, writer)
}
