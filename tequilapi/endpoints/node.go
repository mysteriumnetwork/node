/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// NodeEndpoint struct represents endpoints about node status
type NodeEndpoint struct {
	nodeStatusProvider nodeStatusProvider
}

// NewNodeEndpoint creates and returns node endpoints
func NewNodeEndpoint(nodeStatusProvider nodeStatusProvider) *NodeEndpoint {
	return &NodeEndpoint{
		nodeStatusProvider: nodeStatusProvider,
	}
}

// NodeStatus Status provides Node proposal status
// swagger:operation GET /node/monitoring-status NODE
// ---
// summary: Provides Node proposal status
// description: Node Status as seen by monitoring agent
// responses:
//   200:
//     description: Node status ("passed"/"failed"/"pending)
//     schema:
//       "$ref": "#/definitions/NodeStatusResponse"
func (ne *NodeEndpoint) NodeStatus(c *gin.Context) {
	utils.WriteAsJSON(contract.NodeStatusResponse{Status: ne.nodeStatusProvider.Status()}, c.Writer)
}

// AddRoutesForNode adds nat routes to given router
func AddRoutesForNode(nodeStatusProvider nodeStatusProvider) func(*gin.Engine) error {
	nodeEndpoints := NewNodeEndpoint(nodeStatusProvider)

	return func(e *gin.Engine) error {
		nodeGroup := e.Group("/node")
		{
			nodeGroup.GET("/monitoring-status", nodeEndpoints.NodeStatus)
		}
		return nil
	}
}
