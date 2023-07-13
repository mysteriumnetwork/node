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
	"math/big"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/payments/units"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/launchpad"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

type nodeMonitoringAgent interface {
	Statuses() (node.MonitoringAgentStatuses, error)
	Sessions(rangeTime string) ([]node.SessionItem, error)
	TransferredData(rangeTime string) (node.TransferredData, error)
	SessionsCount(rangeTime string) (node.SessionsCount, error)
	ConsumersCount(rangeTime string) (node.ConsumersCount, error)
	EarningsSeries(rangeTime string) (node.EarningsSeries, error)
	SessionsSeries(rangeTime string) (node.SessionsSeries, error)
	TransferredDataSeries(rangeTime string) (node.TransferredDataSeries, error)
	ProviderActivityStats() (node.ActivityStats, error)
	ProviderQuality() (node.QualityInfo, error)
	EarningsPerService() (node.EarningsPerService, error)
}

// NodeEndpoint struct represents endpoints about node status
type NodeEndpoint struct {
	nodeStatusProvider  nodeStatusProvider
	nodeMonitoringAgent nodeMonitoringAgent
	launchpadAPI        *launchpad.API
}

// NewNodeEndpoint creates and returns node endpoints
func NewNodeEndpoint(nodeStatusProvider nodeStatusProvider, nodeMonitoringAgent nodeMonitoringAgent) *NodeEndpoint {
	return &NodeEndpoint{
		nodeStatusProvider:  nodeStatusProvider,
		nodeMonitoringAgent: nodeMonitoringAgent,
		launchpadAPI:        launchpad.New(),
	}
}

// NodeStatus Status provides Node proposal status
// swagger:operation GET /node/monitoring-status provider NodeStatus
//
//	---
//	summary: Provides Node proposal status
//	description: Node Status as seen by monitoring agent
//	responses:
//	  200:
//	    description: Node status ("passed"/"failed"/"pending)
//	    schema:
//	      "$ref": "#/definitions/NodeStatusResponse"
func (ne *NodeEndpoint) NodeStatus(c *gin.Context) {
	utils.WriteAsJSON(contract.NodeStatusResponse{Status: ne.nodeStatusProvider.Status()}, c.Writer)
}

// MonitoringAgentStatuses Statuses from monitoring agent
// swagger:operation GET /node/monitoring-agent-statuses provider MonitoringAgentStatuses
//
//	---
//	summary: Provides Node connectivity statuses from monitoring agent
//	description: Node connectivity statuses as seen by monitoring agent
//	responses:
//	  200:
//	    description: Monitoring agent statuses ("success"/"cancelled"/"connect_drop/"connect_fail/"internet_fail)
//	    schema:
//	      "$ref": "#/definitions/MonitoringAgentResponse"
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
//
//	---
//	summary: Provides Node sessions data during a period of time
//	description: Node sessions metrics during a period of time
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	    description: Provider sessions list
//	    schema:
//	      "$ref": "#/definitions/ProviderSessionsResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
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
//
//	---
//	summary: Provides total traffic served by the provider during a period of time
//	description: Node transferred data during a period of time
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	    description: Provider transferred data
//	    schema:
//	      "$ref": "#/definitions/ProviderTransferredDataResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
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
//
//	---
//	summary: Provides Node sessions number during a period of time
//	description: Node sessions count during a period of time
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	    description: Provider sessions count
//	    schema:
//	      "$ref": "#/definitions/ProviderSessionsCountResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
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
//
//	---
//	summary: Provides Node consumers number served during a period of time
//	description: Node unique consumers count served during a period of time.
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	   description: Provider consumers count
//	   schema:
//	    "$ref": "#/definitions/ProviderConsumersCountResponse"
//	  400:
//	   description: Failed to parse or request validation failed
//	   schema:
//	    "$ref": "#/definitions/APIError"
//	  500:
//	   description: Internal server error
//	   schema:
//	    "$ref": "#/definitions/APIError"
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

// GetProviderEarningsSeries A time series metrics of earnings during a period of time
// swagger:operation GET /node/provider/series/earnings provider GetProviderEarningsSeries
//
//	---
//	summary: Provides Node  time series metrics of earnings during a period of time
//	description: Node time series metrics of earnings during a period of time.
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	   description: Provider time series metrics of MYSTT earnings
//	   schema:
//	    "$ref": "#/definitions/ProviderEarningsSeriesResponse"
//	  400:
//	   description: Failed to parse or request validation failed
//	   schema:
//	    "$ref": "#/definitions/APIError"
//	  500:
//	   description: Internal server error
//	   schema:
//	    "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderEarningsSeries(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderEarningsSeries))
		return
	}

	res, err := ne.nodeMonitoringAgent.EarningsSeries(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider earnings series: "+err.Error(), contract.ErrorCodeProviderEarningsSeries))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetProviderSessionsSeries A time series metrics of sessions started during a period of time
// swagger:operation GET /node/provider/series/sessions provider GetProviderSessionsSeries
//
//	---
//	summary: Provides Node data series metrics of sessions started during a period of time
//	description: Node time series metrics of sessions started during a period of time.
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	   description: Provider time series metrics of started sessions
//	   schema:
//	    "$ref": "#/definitions/ProviderSessionsSeriesResponse"
//	  400:
//	   description: Failed to parse or request validation failed
//	   schema:
//	    "$ref": "#/definitions/APIError"
//	  500:
//	   description: Internal server error
//	   schema:
//	    "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderSessionsSeries(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderSessionsSeries))
		return
	}

	res, err := ne.nodeMonitoringAgent.SessionsSeries(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider sessions series: "+err.Error(), contract.ErrorCodeProviderSessionsSeries))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetProviderTransferredDataSeries A time series metrics of transferred bytes during a period of time
// swagger:operation GET /node/provider/series/data provider GetProviderTransferredDataSeries
//
//	---
//	summary: Provides Node data series metrics of transferred bytes
//	description: Node data series metrics of transferred bytes during a period of time.
//	parameters:
//	  - in: query
//	    name: range
//	    description: period of time ("1d", "7d", "30d")
//	    type: string
//	responses:
//	  200:
//	   description: Provider time series metrics of transferred bytes
//	   schema:
//	    "$ref": "#/definitions/ProviderTransferredDataSeriesResponse"
//	  400:
//	   description: Failed to parse or request validation failed
//	   schema:
//	    "$ref": "#/definitions/APIError"
//	  500:
//	   description: Internal server error
//	   schema:
//	    "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderTransferredDataSeries(c *gin.Context) {
	rangeTime := c.Query("range")

	switch rangeTime {
	case "1d", "7d", "30d":
	default:
		c.Error(apierror.BadRequest("Invalid time range", contract.ErrorCodeProviderTransferredDataSeries))
		return
	}

	res, err := ne.nodeMonitoringAgent.TransferredDataSeries(rangeTime)
	if err != nil {
		c.Error(apierror.Internal("Could not get provider transferred data series: "+err.Error(), contract.ErrorCodeProviderTransferredDataSeries))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetProviderQuality a quality of provider
// swagger:operation GET /node/provider/quality provider GetProviderQuality
//
//	---
//	summary: Provides Node quality
//	description: Node connectivity quality
//	responses:
//	  200:
//	    description: Provider quality
//	    schema:
//	      "$ref": "#/definitions/QualityInfoResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderQuality(c *gin.Context) {
	res, err := ne.nodeMonitoringAgent.ProviderQuality()
	if err != nil {
		c.Error(apierror.Internal("Could not get provider quality: "+err.Error(), contract.ErrorCodeProviderQuality))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetProviderActivityStats is an activity stats of provider
// swagger:operation GET /node/provider/activity-stats provider GetProviderActivityStats
//
//	---
//	summary: Provides Node activity stats
//	description: Node activity stats
//	responses:
//	  200:
//	    description: Provider activity stats
//	    schema:
//	      "$ref": "#/definitions/ActivityStatsResponse"
//	  400:
//	    description: Failed to parse or request validation failed
//	    schema:
//	      "$ref": "#/definitions/APIError"
//	  500:
//	    description: Internal server error
//	    schema:
//	      "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderActivityStats(c *gin.Context) {

	res, err := ne.nodeMonitoringAgent.ProviderActivityStats()
	if err != nil {
		c.Error(apierror.Internal("Could not get provider activity stats: "+err.Error(), contract.ErrorCodeProviderActivityStats))
		return
	}

	utils.WriteAsJSON(res, c.Writer)
}

// GetLatestRelease retrieves information about the latest node release
// swagger:operation GET /node/latest-release node GetLatestRelease
//
//	---
//	summary: Latest Node release information
//	description: Checks for latest Node release package and retrieves its information
//	responses:
//	  200:
//	   description: Latest Node release information
//	   schema:
//	    "$ref": "#/definitions/LatestReleaseResponse"
//	  500:
//	   description: Failed to retrieve latest Node release information
//	   schema:
//	    "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetLatestRelease(c *gin.Context) {
	version, err := ne.launchpadAPI.LatestPublishedReleaseVersion()
	if err != nil {
		c.Error(apierror.Internal("Could not fetch latest release information", contract.ErrorCodeLatestReleaseInformation))
		log.Error().Err(err).Msg("Could not fetch latest release information")
		return
	}

	utils.WriteAsJSON(contract.LatestReleaseResponse{Version: version}, c.Writer)
}

// GetProviderServiceEarnings Node earnings per service and total earnings in the all network
// swagger:operation GET /node/provider/service-earnings provider GetProviderServiceEarnings
//
//	---
//	summary: Provides Node earnings per service and total earnings in the all network
//	description: Node earnings per service and total earnings in the all network.
//	responses:
//	  200:
//	   description: earnings per service and total earnings
//	   schema:
//	    "$ref": "#/definitions/EarningsPerServiceResponse"
//	  400:
//	   description: Failed to parse or request validation failed
//	   schema:
//	    "$ref": "#/definitions/APIError"
//	  500:
//	   description: Internal server error
//	   schema:
//	    "$ref": "#/definitions/APIError"
func (ne *NodeEndpoint) GetProviderServiceEarnings(c *gin.Context) {
	res, err := ne.nodeMonitoringAgent.EarningsPerService()
	if err != nil {
		c.Error(apierror.Internal("Could not get provider service earnings: "+err.Error(), contract.ErrorCodeProviderServiceEarnings))
		return
	}
	public, _ := strconv.ParseFloat(res.EarningsPublic, 64)
	vpn, _ := strconv.ParseFloat(res.EarningsVPN, 64)
	scraping, _ := strconv.ParseFloat(res.EarningsScraping, 64)
	dvpn, _ := strconv.ParseFloat(res.EarningsDVPN, 64)

	totalPublic, _ := strconv.ParseFloat(res.TotalEarningsPublic, 64)
	totalVPN, _ := strconv.ParseFloat(res.TotalEarningsVPN, 64)
	totalScraping, _ := strconv.ParseFloat(res.TotalEarningsScraping, 64)
	totalDVPN, _ := strconv.ParseFloat(res.TotalEarningsDVPN, 64)

	publicTokens := units.FloatEthToBigIntWei(public)
	vpnTokens := units.FloatEthToBigIntWei(vpn)
	scrapingTokens := units.FloatEthToBigIntWei(scraping)
	dvpnTokens := units.FloatEthToBigIntWei(dvpn)

	totalPublicTokens := units.FloatEthToBigIntWei(totalPublic)
	totalVPNTokens := units.FloatEthToBigIntWei(totalVPN)
	totalScrapingTokens := units.FloatEthToBigIntWei(totalScraping)
	totalDVPNTokens := units.FloatEthToBigIntWei(totalDVPN)

	totalTokens := new(big.Int)
	totalTokens.Add(publicTokens, vpnTokens)
	totalTokens.Add(totalTokens, scrapingTokens)
	totalTokens.Add(totalTokens, dvpnTokens)

	data := contract.EarningsPerServiceResponse{
		EarningsPublic:        contract.NewTokens(publicTokens),
		EarningsVPN:           contract.NewTokens(vpnTokens),
		EarningsScraping:      contract.NewTokens(scrapingTokens),
		EarningsDVPN:          contract.NewTokens(dvpnTokens),
		EarningsTotal:         contract.NewTokens(totalTokens),
		TotalEarningsPublic:   contract.NewTokens(totalPublicTokens),
		TotalEarningsVPN:      contract.NewTokens(totalVPNTokens),
		TotalEarningsScraping: contract.NewTokens(totalScrapingTokens),
		TotalEarningsDVPN:     contract.NewTokens(totalDVPNTokens),
	}

	utils.WriteAsJSON(data, c.Writer)
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
			nodeGroup.GET("/provider/series/earnings", nodeEndpoints.GetProviderEarningsSeries)
			nodeGroup.GET("/provider/series/sessions", nodeEndpoints.GetProviderSessionsSeries)
			nodeGroup.GET("/provider/series/data", nodeEndpoints.GetProviderTransferredDataSeries)
			nodeGroup.GET("/provider/service-earnings", nodeEndpoints.GetProviderServiceEarnings)
			nodeGroup.GET("/latest-release", nodeEndpoints.GetLatestRelease)
			nodeGroup.GET("/provider/quality", nodeEndpoints.GetProviderQuality)
			nodeGroup.GET("/provider/activity-stats", nodeEndpoints.GetProviderActivityStats)
		}
		return nil
	}
}
