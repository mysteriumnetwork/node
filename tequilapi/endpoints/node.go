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
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/go-rest/apierror"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type nodeMonitoringAgent interface {
	Statuses() (node.MonitoringAgentStatuses, error)
	Sessions(rangeTime string) ([]node.SessionItem, error)
	TransferredData(rangeTime string) (node.TransferredData, error)
	SessionsCount(rangeTime string) (node.SessionsCount, error)
	ConsumersCount(rangeTime string) (node.ConsumersCount, error)
	SeriesEarnings(rangeTime string) (node.SeriesEarnings, error)
	SeriesSessions(rangeTime string) (node.SeriesSessions, error)
	SeriesData(rangeTime string) (node.SeriesData, error)
}

// NodeEndpoint struct represents endpoints about node status
type NodeEndpoint struct {
	nodeStatusProvider  nodeStatusProvider
	nodeMonitoringAgent nodeMonitoringAgent
}

// NewNodeEndpoint creates and returns node endpoints
func NewNodeEndpoint(nodeStatusProvider nodeStatusProvider, nodeMonitoringAgent nodeMonitoringAgent) *NodeEndpoint {
	return &NodeEndpoint{
		nodeStatusProvider:  nodeStatusProvider,
		nodeMonitoringAgent: nodeMonitoringAgent,
	}
}

// NodeStatus Status provides Node proposal status
// swagger:operation GET /node/monitoring-status provider NodeStatus
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

// MonitoringAgentStatuses Statuses from monitoring agent
// swagger:operation GET /node/monitoring-agent-statuses provider MonitoringAgentStatuses
// ---
// summary: Provides Node connectivity statuses from monitoring agent
// description: Node connectivity statuses as seen by monitoring agent
// responses:
//   200:
//     description: Monitoring agent statuses ("success"/"cancelled"/"connect_drop/"connect_fail/"internet_fail)
//     schema:
//       "$ref": "#/definitions/MonitoringAgentResponse"
func (ne *NodeEndpoint) MonitoringAgentStatuses(c *gin.Context) {
	res, err := ne.nodeMonitoringAgent.Statuses()
	if err != nil {
		utils.WriteAsJSON(contract.MonitoringAgentResponse{Error: err.Error()}, c.Writer, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(contract.MonitoringAgentResponse{Statuses: res}, c.Writer)
}

// GetProviderSessions A list of sessions metrics during a period of time
// swagger:operation GET /node/provider/sessions provider GetProviderSessions
// ---
// summary: Provides Node sessions data during a period of time
// description: Node sessions metrics during a period of time
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//     description: Provider sessions list
//     schema:
//       "$ref": "#/definitions/ProviderSessionsResponse"
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderSessions(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderSessions))
		return
	}

	res, err := ne.nodeMonitoringAgent.Sessions(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider sessions list: "+err.Error(), contract.ErrorCodeProviderSessions))
		return
	}

	utils.WriteAsJSON(contract.NewProviderSessionsResponse(res), c.Writer)
}

// GetProviderTransferredData A number of bytes transferred during a period of time
// swagger:operation GET /node/provider/transferred-data provider GetProviderTransferredData
// ---
// summary: Provides total traffic served by the provider during a period of time
// description: Node transferred data during a period of time
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//     description: Provider transferred data
//     schema:
//       "$ref": "#/definitions/ProviderTransferredDataResponse"
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderTransferredData(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderTransferredData))
		return
	}

	res, err := ne.nodeMonitoringAgent.TransferredData(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider transferred data: "+err.Error(), contract.ErrorCodeProviderTransferredData))
		return
	}

	utils.WriteAsJSON(contract.ProviderTransferredDataResponse{Bytes: res.Bytes}, c.Writer)
}

// GetProviderSessionsCount A number of sessions during a period of time
// swagger:operation GET /node/provider/sessions-count provider GetProviderSessionsCount
// ---
// summary: Provides Node sessions number during a period of time
// description: Node sessions count during a period of time
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//     description: Provider sessions count
//     schema:
//       "$ref": "#/definitions/ProviderSessionsCountResponse"
//   400:
//     description: Failed to parse or request validation failed
//     schema:
//       "$ref": "#/definitions/APIError"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderSessionsCount(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderSessionsCount))
		return
	}

	res, err := ne.nodeMonitoringAgent.SessionsCount(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider sessions count: "+err.Error(), contract.ErrorCodeProviderSessionsCount))
		return
	}

	utils.WriteAsJSON(contract.ProviderSessionsCountResponse{Count: res.Count}, c.Writer)
}

// GetProviderConsumersCount A number of consumers served during a period of time
// swagger:operation GET /node/provider/consumers-count provider GetProviderConsumersCount
// ---
// summary: Provides Node consumers number served during a period of time
// description: Node unique consumers count served during a period of time.
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//    description: Provider consumers count
//    schema:
//     "$ref": "#/definitions/ProviderConsumersCountResponse"
//   400:
//    description: Failed to parse or request validation failed
//    schema:
//     "$ref": "#/definitions/APIError"
//   500:
//    description: Internal server error
//    schema:
//     "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderConsumersCount(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderConsumersCount))
		return
	}

	res, err := ne.nodeMonitoringAgent.ConsumersCount(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider consumers count: "+err.Error(), contract.ErrorCodeProviderConsumersCount))
		return
	}

	utils.WriteAsJSON(contract.ProviderConsumersCountResponse{Count: res.Count}, c.Writer)
}

// GetProviderSeriesEarnings A time series metrics of earnings during a period of time
// swagger:operation GET /node/provider/series/earnings provider GetProviderSeriesEarnings
// ---
// summary: Provides Node  time series metrics of earnings during a period of time
// description: Node time series metrics of earnings during a period of time.
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//    description: Provider time series metrics of MYSTT earnings
//    schema:
//     "$ref": "#/definitions/ProviderSeriesEarningsResponse"
//   400:
//    description: Failed to parse or request validation failed
//    schema:
//     "$ref": "#/definitions/APIError"
//   500:
//    description: Internal server error
//    schema:
//     "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderSeriesEarnings(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderSeriesEarnings))
		return
	}

	res, err := ne.nodeMonitoringAgent.SeriesEarnings(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider consumers count: "+err.Error(), contract.ErrorCodeProviderSeriesEarnings))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetProviderSeriesSessions A time series metrics of sessions started during a period of time
// swagger:operation GET /node/provider/series/sessions provider GetProviderSeriesSessions
// ---
// summary: Provides Node data series metrics of sessions started during a period of time
// description: Node time series metrics of sessions started during a period of time.
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//    description: Provider time series metrics of started sessions
//    schema:
//     "$ref": "#/definitions/ProviderSeriesSessionsResponse"
//   400:
//    description: Failed to parse or request validation failed
//    schema:
//     "$ref": "#/definitions/APIError"
//   500:
//    description: Internal server error
//    schema:
//     "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderSeriesSessions(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderSeriesSessions))
		return
	}

	res, err := ne.nodeMonitoringAgent.SeriesSessions(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider consumers count: "+err.Error(), contract.ErrorCodeProviderSeriesSessions))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetProviderSeriesData A time series metrics of transferred bytes during a period of time
// swagger:operation GET /node/provider/series/data provider GetProviderSeriesData
// ---
// summary: Provides Node data series metrics of transferred bytes
// description: Node data series metrics of transferred bytes during a period of time.
// parameters:
//   - in: query
//     name: range
//     description: period of time ("1d", "7d", "30d")
//     type: string
// responses:
//   200:
//    description: Provider time series metrics of transferred bytes
//    schema:
//     "$ref": "#/definitions/ProviderSeriesDataResponse"
//   400:
//    description: Failed to parse or request validation failed
//    schema:
//     "$ref": "#/definitions/APIError"
//   500:
//    description: Internal server error
//    schema:
//     "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderSeriesData(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderSeriesData))
		return
	}

	res, err := ne.nodeMonitoringAgent.SeriesData(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider consumers count: "+err.Error(), contract.ErrorCodeProviderSeriesData))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// AddRoutesForNode adds nat routes to given router
func AddRoutesForNode(nodeStatusProvider nodeStatusProvider, nodeMonitoringAgent nodeMonitoringAgent) func(*gin.Engine) error {
	nodeEndpoints := NewNodeEndpoint(nodeStatusProvider, nodeMonitoringAgent)

	return func(e *gin.Engine) error {
		nodeGroup := e.Group("/node")
		{
			nodeGroup.GET("/monitoring-status", nodeEndpoints.NodeStatus)
			nodeGroup.GET("/monitoring-agent-statuses", nodeEndpoints.MonitoringAgentStatuses)
			nodeGroup.GET("/provider/sessions", nodeEndpoints.GetProviderSessions)
			nodeGroup.GET("/provider/transferred-data", nodeEndpoints.GetProviderTransferredData)
			nodeGroup.GET("/provider/sessions-count", nodeEndpoints.GetProviderSessionsCount)
			nodeGroup.GET("/provider/consumers-count", nodeEndpoints.GetProviderConsumersCount)
			nodeGroup.GET("/provider/series/earnings", nodeEndpoints.GetProviderSeriesEarnings)
			nodeGroup.GET("/provider/series/sessions", nodeEndpoints.GetProviderSeriesSessions)
			nodeGroup.GET("/provider/series/data", nodeEndpoints.GetProviderSeriesData)
		}
		return nil
	}
}
