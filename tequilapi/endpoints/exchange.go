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
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
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
//
//	---
//	summary: Returns the myst price in the given currency
//	description: Returns the myst price in the given currency (dai is deprecated)
//	parameters:
//	  - name: currency
//	    in: path
//	    description: Currency to which MYST is converted
//	    type: string
//	    required: true
//	responses:
//	  200:
//	    description: MYST price in given currency
//	    schema:
//	      "$ref": "#/definitions/CurrencyExchangeDTO"
//	  404:
//	    description: Currency is not supported
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (e *exchangeEndpoint) ExchangeMyst(c *gin.Context) {
	currency := strings.ToUpper(c.Param("currency"))

	rates, err := e.me.GetMystExchangeRate()
	if err != nil {
		c.Error(err)
		return
	}

	var ok bool
	amount, ok := rates[currency]
	if !ok {
		c.Error(apierror.NotFound("Currency is not supported"))
		return
	}

	rate := contract.CurrencyExchangeDTO{
		Amount:   amount,
		Currency: currency,
	}

	utils.WriteAsJSON(rate, c.Writer)
}

// AddRoutesForCurrencyExchange attaches exchange endpoints to router.
func AddRoutesForCurrencyExchange(mystex mystexchange) func(*gin.Engine) error {
	e := NewExchangeEndpoint(mystex)
	return func(g *gin.Engine) error {
		ex := g.Group("/exchange")
		{
			ex.GET("/myst/:currency", e.ExchangeMyst)
		}
		return nil
	}
}
