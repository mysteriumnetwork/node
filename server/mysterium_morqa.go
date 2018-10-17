/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package server

import (
	"net/http"
	"time"

	log "github.com/cihub/seelog"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/server/dto"
)

const (
	mysteriumMorqaLogPrefix = "[Mysterium.morqa] "
)

type mysteriumMorqa struct {
	http                 HTTPTransport
	qualityOracleAddress string
}

// NewMorqaClient creates Mysterium Morqa client with a real communication
func NewMorqaClient(qualityOracleAddress string) MorqaClient {
	return &mysteriumMorqa{
		newHTTPTransport(1 * time.Minute),
		qualityOracleAddress,
	}
}

// ProposalsQuality returns a list of proposals connection quality
func (m *mysteriumMorqa) ProposalsQuality() ([]dto.QualityConnects, error) {
	req, err := requests.NewGetRequest(m.qualityOracleAddress, "proposals/quality", nil)
	if err != nil {
		return nil, err
	}

	var qualityResponse dto.ServiceQualityResponse
	err = m.doRequestAndParseResponse(req, &qualityResponse)
	if err != nil {
		return nil, err
	}

	return qualityResponse.Connects, nil
}

func (m *mysteriumMorqa) doRequestAndParseResponse(req *http.Request, responseValue interface{}) error {
	resp, err := m.http.Do(req)
	if err != nil {
		log.Error(mysteriumMorqaLogPrefix, err)
		return err
	}
	defer resp.Body.Close()

	err = parseResponseError(resp)
	if err != nil {
		log.Error(mysteriumMorqaLogPrefix, err)
		return err
	}

	return parseResponseJSON(resp, responseValue)
}
