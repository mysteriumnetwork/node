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
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
	"github.com/pkg/errors"
)

type api interface {
	GetPaymentOrder(id identity.Identity, oid uint64) (*pilvytis.OrderResponse, error)
	GetPaymentOrders(id identity.Identity) ([]pilvytis.OrderResponse, error)
	GetPaymentOrderCurrencies() ([]string, error)
	GetPaymentOrderOptions() (*pilvytis.PaymentOrderOptions, error)
}

type paymentsIssuer interface {
	CreatePaymentOrder(id identity.Identity, mystAmount float64, payCurrency string, lightning bool) (*pilvytis.OrderResponse, error)
}

type pilvytisEndpoint struct {
	api api
	pt  paymentsIssuer
}

// NewPilvytisEndpoint returns pilvytis endpoints.
func NewPilvytisEndpoint(pil api, pt paymentsIssuer) *pilvytisEndpoint {
	return &pilvytisEndpoint{
		api: pil,
		pt:  pt,
	}
}

// CreatePaymentOrder creates a new payment order.
//
// swagger:operation POST /identities/{id}/payment-order Order createOrder
// ---
// summary: Create order
// description: Takes the given data and tries to create a new payment order in the pilvytis service.
// parameters:
// - name: id
//   in: path
//   description: Identity for which to create an order
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Required data to create a new order
//   schema:
//     $ref: "#/definitions/OrderRequest"
// responses:
//   200:
//     description: Order object
//     schema:
//       "$ref": "#/definitions/OrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *pilvytisEndpoint) CreatePaymentOrder(c *gin.Context) {
	r := c.Request
	w := c.Writer
	params := c.Params

	var req contract.OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.SendError(w, errors.Wrap(err, "failed to parse order req"), http.StatusBadRequest)
		return
	}

	rid := identity.FromAddress(params.ByName("id"))
	resp, err := e.pt.CreatePaymentOrder(
		rid,
		req.MystAmount,
		req.PayCurrency,
		req.LightningNetwork)
	if err != nil {
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.NewOrderResponse(*resp), w)
}

// GetPaymentOrder returns a payment order which maches a given ID and identity.
//
// swagger:operation GET /identities/{id}/payment-order/{order_id} Order getOrder
// ---
// summary: Get order
// description: Gets an order for a given identity and order id combo from the pilvytis service
// parameters:
// - name: id
//   in: path
//   description: Identity for which to get an order
//   type: string
//   required: true
// - name: order_id
//   in: path
//   description: Order id
//   type: integer
//   required: true
// responses:
//   200:
//     description: Order object
//     schema:
//       "$ref": "#/definitions/OrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *pilvytisEndpoint) GetPaymentOrder(c *gin.Context) {
	w := c.Writer
	params := c.Params

	id := params.ByName("order_id")
	if id == "" {
		utils.SendError(w, errors.New("missing ID param"), http.StatusBadRequest)
		return
	}

	orderID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		utils.SendError(w, errors.New("can't parse order ID as uint"), http.StatusBadRequest)
		return
	}

	resp, err := e.api.GetPaymentOrder(identity.FromAddress(params.ByName("id")), orderID)
	if err != nil {
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.NewOrderResponse(*resp), w)
}

// GetPaymentOrder returns a payment order which maches a given ID and identity.
//
// swagger:operation GET /identities/{id}/payment-order Order getOrders
// ---
// summary: Get all orders for identity
// description: Gets all orders for a given identity from the pilvytis service
// parameters:
// - name: id
//   in: path
//   description: Identity for which to get orders
//   type: string
//   required: true
// responses:
//   200:
//     description: Array of order objects
//     schema:
//       type: "array"
//       items:
//         "$ref": "#/definitions/OrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *pilvytisEndpoint) GetPaymentOrders(c *gin.Context) {
	w := c.Writer
	params := c.Params

	resp, err := e.api.GetPaymentOrders(identity.FromAddress(params.ByName("id")))
	if err != nil {
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.NewOrdersResponse(resp), w)
}

// GetPaymentOrderCurrencies returns a slice of possible order currencies
//
// swagger:operation GET /payment-order-currencies Order getOrdersCurrencies
// ---
// summary: Get all possible currencies for payments
// description: Gets all possible currencies that can be used for payments
// responses:
//   200:
//     description: Array of order objects
//     schema:
//       type: "array"
//       items:
//         type: string
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *pilvytisEndpoint) GetPaymentOrderCurrencies(c *gin.Context) {
	w := c.Writer

	resp, err := e.api.GetPaymentOrderCurrencies()
	if err != nil {
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(resp, w)
}

// GetPaymentOrderOptions returns recommendation on myst amounts
//
// swagger:operation GET /payment-order-options Order GetPaymentOrderOptions
// ---
// summary: Get payment order options
// description: Includes minimum amount of myst required when topping up and suggested amounts
// responses:
//   200:
//     description: PaymentOrderOptions object
//     schema:
//       "$ref": "#/definitions/PaymentOrderOptions"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (e *pilvytisEndpoint) GetPaymentOrderOptions(c *gin.Context) {
	w := c.Writer

	resp, err := e.api.GetPaymentOrderOptions()
	if err != nil {
		utils.SendError(w, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.ToPaymentOrderOptions(resp), w)
}

// AddRoutesForPilvytis adds the pilvytis routers to the given router.
func AddRoutesForPilvytis(pilvytis api, pt paymentsIssuer) func(*gin.Engine) error {
	pil := NewPilvytisEndpoint(pilvytis, pt)
	return func(e *gin.Engine) error {
		idGroup := e.Group("/identities")
		{
			idGroup.POST("/:id/payment-order", pil.CreatePaymentOrder)
			idGroup.GET("/:id/payment-order/:order_id", pil.GetPaymentOrder)
			idGroup.GET("/:id/payment-order", pil.GetPaymentOrders)
		}
		e.GET("/payment-order-options", pil.GetPaymentOrderOptions)
		e.GET("/payment-order-currencies", pil.GetPaymentOrderCurrencies)
		return nil
	}
}
