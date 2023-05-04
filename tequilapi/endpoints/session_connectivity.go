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
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// swagger:model ConnectivityStatus
type sessionConnectivityStatusCollection struct {
	Entries []*sessionConnectivityStatus `json:"entries"`
}

type sessionConnectivityStatus struct {
	PeerAddress  string    `json:"peer_address"`
	SessionID    string    `json:"session_id"`
	Code         uint32    `json:"code"`
	Message      string    `json:"message"`
	CreatedAtUTC time.Time `json:"created_at_utc"`
}

type sessionConnectivityEndpoint struct {
	statusStorage connectivity.StatusStorage
}

// swagger:operation GET /sessions-connectivity-status ConnectivityStatus
//
//	---
//	summary: Returns session connectivity status
//	description: Returns list of session connectivity status
//	responses:
//	  200:
//	    description: List of connectivity statuses
//	    schema:
//	      "$ref": "#/definitions/ConnectivityStatus"
func (e *sessionConnectivityEndpoint) List(c *gin.Context) {
	r := sessionConnectivityStatusCollection{
		Entries: []*sessionConnectivityStatus{},
	}

	for _, entry := range e.statusStorage.GetAllStatusEntries() {
		r.Entries = append(r.Entries, &sessionConnectivityStatus{
			PeerAddress:  entry.PeerID.Address,
			SessionID:    entry.SessionID,
			Code:         uint32(entry.StatusCode),
			Message:      entry.Message,
			CreatedAtUTC: entry.CreatedAtUTC,
		})
	}

	utils.WriteAsJSON(r, c.Writer)
}

// AddRoutesForConnectivityStatus attaches connectivity statuses endpoints to router.
func AddRoutesForConnectivityStatus(statusStorage connectivity.StatusStorage) func(*gin.Engine) error {
	e := &sessionConnectivityEndpoint{
		statusStorage: statusStorage,
	}
	return func(g *gin.Engine) error {
		g.GET("/sessions-connectivity-status", e.List)
		return nil
	}
}
