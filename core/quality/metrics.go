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

package quality

import (
	"encoding/json"
)

// ServiceMetricsResponse represents response from the quality oracle service
type ServiceMetricsResponse struct {
	Connects []json.RawMessage `json:"connects"`
}

// Parse parses JSON metrics message to the proposal, and return JSON with metrics only
func Parse(msg json.RawMessage, proposal interface{}) ([]byte, error) {
	var metrics struct {
		ConnectCount json.RawMessage `json:"connectCount"`
	}

	if err := json.Unmarshal(msg, &proposal); err != nil {
		log.Warn("failed to parse proposal info")
		return nil, err
	}

	if err := json.Unmarshal(msg, &metrics); err != nil {
		log.Warn("failed to parse metrics")
		return nil, err
	}

	out, err := json.Marshal(metrics)
	if err != nil {
		log.Warn("failed to marshal metrics JSON")
		return nil, err
	}
	return out, err
}
