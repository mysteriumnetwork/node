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
	"strconv"
	"strings"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"

	"github.com/gin-gonic/gin"
)

type api interface {
	// TODO: Deprecated
	// ================
	GetPaymentOrder(id identity.Identity, oid uint64) (*pilvytis.OrderResponse, error)
	GetPaymentOrders(id identity.Identity) ([]pilvytis.OrderResponse, error)
	GetPaymentOrderCurrencies() ([]string, error)
	GetPaymentOrderOptions() (*pilvytis.PaymentOrderOptions, error)
	// =================

	GetPaymentGatewayOrder(id identity.Identity, oid string) (*pilvytis.GatewayOrderResponse, error)
	GetPaymentGatewayOrderInvoice(id identity.Identity, oid string) ([]byte, error)
	GetPaymentGatewayOrders(id identity.Identity) ([]pilvytis.GatewayOrderResponse, error)
	GetPaymentGateways() ([]pilvytis.GatewaysResponse, error)
	GetRegistrationPaymentStatus(id identity.Identity) (*pilvytis.RegistrationPaymentResponse, error)
}

type paymentsIssuer interface {
	CreatePaymentOrder(id identity.Identity, mystAmount float64, payCurrency string, lightning bool) (*pilvytis.OrderResponse, error)
	CreatePaymentGatewayOrder(cgo pilvytis.GatewayOrderRequest) (*pilvytis.GatewayOrderResponse, error)
}

type paymentLocationFallback interface {
	GetOrigin() locationstate.Location
}

type pilvytisEndpoint struct {
	api api
	pt  paymentsIssuer
	lf  paymentLocationFallback
}

// NewPilvytisEndpoint returns pilvytis endpoints.
func NewPilvytisEndpoint(pil api, pt paymentsIssuer, lf paymentLocationFallback) *pilvytisEndpoint {
	return &pilvytisEndpoint{
		api: pil,
		pt:  pt,
		lf:  lf,
	}
}

// CreatePaymentOrder creates a new payment order.
//
// swagger:operation POST /identities/{id}/payment-order Order createOrder
// ---
// summary: Create order
// description: Takes the given data and tries to create a new payment order in the pilvytis service.
// deprecated: true
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
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) CreatePaymentOrder(c *gin.Context) {
	var req contract.OrderRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	rid := identity.FromAddress(c.Param("id"))
	resp, err := e.pt.CreatePaymentOrder(
		rid,
		req.MystAmount,
		req.PayCurrency,
		req.LightningNetwork)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to create payment order", contract.ErrCodePaymentCreate))
		return
	}

	utils.WriteAsJSON(contract.NewOrderResponse(*resp), c.Writer)
}

// GetPaymentOrder returns a payment order which maches a given ID and identity.
//
// swagger:operation GET /identities/{id}/payment-order/{order_id} Order getOrder
// ---
// summary: Get order
// description: Gets an order for a given identity and order id combo from the pilvytis service
// deprecated: true
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
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentOrder(c *gin.Context) {
	id := c.Param("order_id")
	if id == "" {
		c.Error(apierror.BadRequestField("'order_id' is required", apierror.ValidateErrRequired, "order_id"))
		return
	}

	orderID, err := strconv.ParseUint(id, 10, 64)
	if err != nil {
		c.Error(apierror.BadRequestField("'order_id' is invalid", apierror.ValidateErrInvalidVal, "order_id"))
		return
	}

	resp, err := e.api.GetPaymentOrder(identity.FromAddress(c.Param("id")), orderID)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get order", contract.ErrCodePaymentGet))
		return
	}

	utils.WriteAsJSON(contract.NewOrderResponse(*resp), c.Writer)
}

// GetPaymentOrders returns a payment order which maches a given ID and identity.
//
// swagger:operation GET /identities/{id}/payment-order Order getOrders
// ---
// summary: Get all orders for identity
// description: Gets all orders for a given identity from the pilvytis service
// deprecated: true
// parameters:
// - name: id
//   in: path
//   description: Identity for which to get orders
//   type: string
//   required: true
// responses:
//   200:
//     description: List of orders
//     schema:
//       type: "array"
//       items:
//         "$ref": "#/definitions/OrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentOrders(c *gin.Context) {
	resp, err := e.api.GetPaymentOrders(identity.FromAddress(c.Param("id")))
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get orders", contract.ErrCodePaymentList))
		return
	}

	utils.WriteAsJSON(contract.NewOrdersResponse(resp), c.Writer)
}

// GetPaymentOrderCurrencies returns a slice of possible order currencies
//
// swagger:operation GET /payment-order-currencies Order getOrdersCurrencies
// ---
// summary: Get all possible currencies for payments
// description: Gets all possible currencies that can be used for payments
// deprecated: true
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
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentOrderCurrencies(c *gin.Context) {
	resp, err := e.api.GetPaymentOrderCurrencies()
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get currencies", contract.ErrCodePaymentListCurrencies))
		return
	}

	utils.WriteAsJSON(resp, c.Writer)
}

// GetPaymentOrderOptions returns recommendation on myst amounts
//
// swagger:operation GET /payment-order-options Order GetPaymentOrderOptions
// ---
// summary: Get payment order options
// description: Includes minimum amount of myst required when topping up and suggested amounts
// deprecated: true
// responses:
//   200:
//     description: Amount options for creating a payment order
//     schema:
//       "$ref": "#/definitions/PaymentOrderOptions"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentOrderOptions(c *gin.Context) {
	resp, err := e.api.GetPaymentOrderOptions()
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get order options", contract.ErrCodePaymentGetOptions))
		return
	}

	utils.WriteAsJSON(contract.ToPaymentOrderOptions(resp), c.Writer)
}

// GetPaymentGateways returns data about supported payment gateways.
//
// swagger:operation GET /v2/payment-order-gateways Order getPaymentGateways
// ---
// summary: Get payment gateway configuration.
// description: Returns gateway configuration including supported currencies, minimum amounts, etc.
// responses:
//   200:
//     description: List of payment gateways
//     schema:
//       type: "array"
//       items:
//         "$ref": "#/definitions/GatewaysResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentGateways(c *gin.Context) {
	resp, err := e.api.GetPaymentGateways()
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to list payment gateways", contract.ErrCodePaymentListGateways))
		return
	}

	utils.WriteAsJSON(contract.ToGatewaysReponse(resp), c.Writer)
}

// GetPaymentGatewayOrders returns a list of payment orders.
//
// swagger:operation GET /v2/identities/{id}/payment-order Order getPaymentGatewayOrders
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
//     description: List of payment orders
//     schema:
//       type: "array"
//       items:
//         "$ref": "#/definitions/PaymentOrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentGatewayOrders(c *gin.Context) {
	resp, err := e.api.GetPaymentGatewayOrders(identity.FromAddress(c.Param("id")))
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to list orders", contract.ErrCodePaymentList))
		return
	}

	utils.WriteAsJSON(contract.NewPaymentOrdersResponse(resp), c.Writer)
}

// GetPaymentGatewayOrder returns a payment order which maches a given ID and identity.
//
// swagger:operation GET /v2/identities/{id}/payment-order/{order_id} Order getPaymentGatewayOrder
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
//       "$ref": "#/definitions/PaymentOrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentGatewayOrder(c *gin.Context) {
	resp, err := e.api.GetPaymentGatewayOrder(
		identity.FromAddress(c.Param("id")),
		c.Param("order_id"),
	)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get payment order", contract.ErrCodePaymentGet))
		return
	}

	utils.WriteAsJSON(contract.NewPaymentOrderResponse(resp), c.Writer)
}

// GetPaymentGatewayOrderInvoice returns an invoice for payment order matching the given ID and identity.
//
// swagger:operation GET /v2/identities/{id}/payment-order/{order_id}/invoice Order getPaymentGatewayOrderInvoice
// ---
// summary: Get invoice
// description: Gets an invoice for payment order matching the given ID and identity
// parameters:
// - name: id
//   in: path
//   description: Identity for which to get an order invoice
//   type: string
//   required: true
// - name: order_id
//   in: path
//   description: Order id
//   type: integer
//   required: true
// responses:
//   200:
//     description: Invoice PDF (binary)
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetPaymentGatewayOrderInvoice(c *gin.Context) {
	resp, err := e.api.GetPaymentGatewayOrderInvoice(
		identity.FromAddress(c.Param("id")),
		c.Param("order_id"),
	)
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get invoice", contract.ErrCodePaymentGetInvoice))
		return
	}

	c.Data(200, "application/pdf", resp)
}

// CreatePaymentGatewayOrder creates a new payment order.
//
// swagger:operation POST /identities/{id}/{gw}/payment-order Order createPaymentGatewayOrder
// ---
// summary: Create order
// description: Takes the given data and tries to create a new payment order in the pilvytis service.
// parameters:
// - name: id
//   in: path
//   description: Identity for which to create an order
//   type: string
//   required: true
// - name: gw
//   in: path
//   description: Gateway for which a payment order is created
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Required data to create a new order
//   schema:
//     $ref: "#/definitions/PaymentOrderRequest"
// responses:
//   200:
//     description: Payment order
//     schema:
//       "$ref": "#/definitions/PaymentOrderResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) CreatePaymentGatewayOrder(c *gin.Context) {
	var req contract.PaymentOrderRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	// TODO: Remove this once apps update
	if req.Country == "" {
		org := e.lf.GetOrigin()
		req.Country = strings.ToUpper(org.Country)
	}

	rid := identity.FromAddress(c.Param("id"))

	resp, err := e.pt.CreatePaymentGatewayOrder(req.GatewayOrderRequest(rid, c.Param("gw")))
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to create payment order", contract.ErrCodePaymentCreate))
		return
	}

	utils.WriteAsJSON(contract.NewPaymentOrderResponse(resp), c.Writer)
}

// GetRegistrationPaymentStatus returns a whether a registration order has been paid.
//
// swagger:operation GET /v2/identities/{id}/registration-payment Order getRegistrationPaymentStatus
// ---
// summary: Check for registration payment
// description: Checks if a registration payment order for an identity has been paid in pilvytis.
// parameters:
// - name: id
//   in: path
//   description: Identity for which to check
//   type: string
//   required: true
// responses:
//   200:
//     description: Registration order status
//     schema:
//       "$ref": "#/definitions/RegistrationPaymentResponse"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (e *pilvytisEndpoint) GetRegistrationPaymentStatus(c *gin.Context) {
	resp, err := e.api.GetRegistrationPaymentStatus(identity.FromAddress(c.Param("id")))
	if err != nil {
		utils.ForwardError(c, err, apierror.Internal("Failed to get registration payment status", contract.ErrCodePaymentList))
		return
	}

	utils.WriteAsJSON(contract.NewRegistrationPaymentResponse(resp), c.Writer)
}

// AddRoutesForPilvytis adds the pilvytis routers to the given router.
func AddRoutesForPilvytis(pilvytis api, pt paymentsIssuer, lf paymentLocationFallback) func(*gin.Engine) error {
	pil := NewPilvytisEndpoint(pilvytis, pt, lf)
	return func(e *gin.Engine) error {
		// TODO: Deprecated
		// =====
		idGroup := e.Group("/identities")
		{
			idGroup.POST("/:id/payment-order", pil.CreatePaymentOrder)
			idGroup.GET("/:id/payment-order/:order_id", pil.GetPaymentOrder)
			idGroup.GET("/:id/payment-order", pil.GetPaymentOrders)
		}
		e.GET("/payment-order-options", pil.GetPaymentOrderOptions)
		e.GET("/payment-order-currencies", pil.GetPaymentOrderCurrencies)
		// =====

		idGroupV2 := e.Group("/v2/identities")
		{
			idGroupV2.POST("/:id/:gw/payment-order", pil.CreatePaymentGatewayOrder)
			idGroupV2.GET("/:id/payment-order/:order_id", pil.GetPaymentGatewayOrder)
			idGroupV2.GET("/:id/payment-order/:order_id/invoice", pil.GetPaymentGatewayOrderInvoice)
			idGroupV2.GET("/:id/payment-order", pil.GetPaymentGatewayOrders)
			idGroupV2.GET("/:id/registration-payment", pil.GetRegistrationPaymentStatus)
		}
		e.GET("/v2/payment-order-gateways", pil.GetPaymentGateways)
		return nil
	}
}
