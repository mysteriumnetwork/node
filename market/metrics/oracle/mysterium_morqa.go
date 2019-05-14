/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package oracle

import (
	"encoding/json"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/market/metrics"
	"github.com/mysteriumnetwork/node/requests"
)

const (
	mysteriumMorqaLogPrefix = "[Mysterium.morqa] "
)

type mysteriumMorqa struct {
	http                 requests.HTTPTransport
	qualityOracleAddress string
}

// NewMorqaClient creates Mysterium Morqa client with a real communication
func NewMorqaClient(qualityOracleAddress string) metrics.QualityOracle {
	return &mysteriumMorqa{
		requests.NewHTTPClient(1 * time.Minute),
		qualityOracleAddress,
	}
}

// ProposalsMetrics returns a list of proposals connection metrics
func (m *mysteriumMorqa) ProposalsMetrics() []json.RawMessage {
	req, err := requests.NewGetRequest(m.qualityOracleAddress, "proposals/quality", nil)
	if err != nil {
		log.Warn(mysteriumMorqaLogPrefix, "Failed to create proposals metrics request: ", err)
		return nil
	}

	var metricsResponse metrics.ServiceMetricsResponse
	err = m.http.DoRequestAndParseResponse(req, &metricsResponse)
	if err != nil {
		log.Warn(mysteriumMorqaLogPrefix, "Failed to request or parse proposals metrics: ", err)
		return nil
	}

	return metricsResponse.Connects
}
