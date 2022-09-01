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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

type mockNodeStatusProvider struct {
	status node.MonitoringStatus
}

type mockMonitoringAgent struct {
	status         node.MonitoringAgentStatuses
	sessions       []node.SessionItem
	data           node.TransferredData
	sessionsCount  node.SessionsCount
	consumersCount node.ConsumersCount
	seriesEarnings node.SeriesEarnings
	seriesSessions node.SeriesSessions
	seriesData     node.SeriesData
}

func (nodeStatusTracker *mockNodeStatusProvider) Status() node.MonitoringStatus {
	return nodeStatusTracker.status
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) Statuses() (node.MonitoringAgentStatuses, error) {
	return nodeMonitoringAgentTracker.status, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) Sessions(_ string) ([]node.SessionItem, error) {
	return nodeMonitoringAgentTracker.sessions, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) TransferredData(_ string) (node.TransferredData, error) {
	return nodeMonitoringAgentTracker.data, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) SessionsCount(_ string) (node.SessionsCount, error) {
	return nodeMonitoringAgentTracker.sessionsCount, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) ConsumersCount(_ string) (node.ConsumersCount, error) {
	return nodeMonitoringAgentTracker.consumersCount, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) SeriesEarnings(_ string) (node.SeriesEarnings, error) {
	return nodeMonitoringAgentTracker.seriesEarnings, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) SeriesSessions(_ string) (node.SeriesSessions, error) {
	return nodeMonitoringAgentTracker.seriesSessions, nil
}

func (nodeMonitoringAgentTracker *mockMonitoringAgent) SeriesData(_ string) (node.SeriesData, error) {
	return nodeMonitoringAgentTracker.seriesData, nil
}

func Test_NodeStatus(t *testing.T) {
	// given:
	mockStatusTracker := &mockNodeStatusProvider{}
	mockMonitoringAgentTracker := &mockMonitoringAgent{}

	router := gin.Default()
	err := AddRoutesForNode(mockStatusTracker, mockMonitoringAgentTracker)(router)
	assert.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, "/node/monitoring-status", nil)
	assert.Nil(t, err)

	// expect:
	for _, status := range []node.MonitoringStatus{
		"passed",
		"failed",
		"pending",
	} {
		t.Run(fmt.Sprintf("Consumer receives node status: %s", status), func(t *testing.T) {
			resp := httptest.NewRecorder()
			mockStatusTracker.status = status
			router.ServeHTTP(resp, req)

			result, err := json.Marshal(contract.NodeStatusResponse{Status: status})
			assert.NoError(t, err)
			assert.JSONEq(t, string(result), resp.Body.String())
		})
	}
}
