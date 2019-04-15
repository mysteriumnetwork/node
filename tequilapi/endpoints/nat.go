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

package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/nat/traversal"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model NatStatusDTO
type NatStatusDTO struct {
	Successful bool   `json:"successful"`
	Error      string `json:"error"`
}

type NatEvents interface {
	LastEvent() *traversal.Event
}

// ServiceEndpoint struct represents management of service resource and it's sub-resources
type NatEndpoint struct {
	natEvents NatEvents
}

// NewNatEndpoint creates and returns nat endpoint
func NewNatEndpoint(natEvents NatEvents) *NatEndpoint {
	return &NatEndpoint{
		natEvents: natEvents,
	}
}

// NatStatus provides NAT configuration info
// swagger:operation GET /nat/status Nat NatStatusDTO
// ---
// summary: Shows NAT status
// description: Nat status returns the last known NAT event
// responses:
//   200:
//     description: NAT status and/or error
//     schema:
//       "$ref": "#/definitions/NatStatusDTO"
//   204:
//     description: No status available
func (ne *NatEndpoint) NatStatus(resp http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	event := ne.natEvents.LastEvent()

	statusResponse := toNatStatusResponse(event)
	if statusResponse == nil {
		utils.SendErrorMessage(resp, "No status is available", http.StatusNoContent)
		return
	}

	utils.WriteAsJSON(statusResponse, resp)
}

// AddRoutesForService adds service routes to given router
func AddRoutesForNat(router *httprouter.Router, natEvents NatEvents) {
	natEndpoint := NewNatEndpoint(natEvents)

	router.GET("/nat/status", natEndpoint.NatStatus)
}

func toNatStatusResponse(event *traversal.Event) *NatStatusDTO {
	if event == nil {
		return nil
	}

	status := event.Successful
	var error string
	if event.Error != nil {
		error = event.Error.Error()
	}
	return &NatStatusDTO{
		Successful: status,
		Error:      error,
	}
}
