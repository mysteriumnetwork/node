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

package event

import (
	"time"

	"github.com/mysteriumnetwork/node/market"
)

// AppTopicState is the topic that we use to announce state changes to via the event bus
const AppTopicState = "State change"

// State represents the node state at the current moment. It's a read only object, used only to display data.
type State struct {
	NATStatus NATStatus        `json:"natStatus"`
	Services  []ServiceInfo    `json:"serviceInfo"`
	Sessions  []ServiceSession `json:"sessions"`
}

// NATStatus stores the nat status related information
// swagger:model NATStatusDTO
type NATStatus struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// ConnectionStatistics shows the successful and attempted connection count
type ConnectionStatistics struct {
	Attempted  int `json:"attempted"`
	Successful int `json:"successful"`
}

// ServiceInfo stores the information about a service
type ServiceInfo struct {
	ID                   string                 `json:"id"`
	ProviderID           string                 `json:"providerId"`
	Type                 string                 `json:"type"`
	Options              interface{}            `json:"options"`
	Status               string                 `json:"status"`
	Proposal             market.ServiceProposal `json:"proposal"`
	AccessPolicies       *[]market.AccessPolicy `json:"accessPolicies,omitempty"`
	Sessions             []ServiceSession       `json:"serviceSession,omitempty"`
	ConnectionStatistics ConnectionStatistics   `json:"connectionStatistics"`
}

// ServiceSession represents the session object
// swagger:model ServiceSessionDTO
type ServiceSession struct {
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ID string `json:"id"`
	// example: 0x0000000000000000000000000000000000000001
	ConsumerID string `json:"consumerId"`
	// example: 2019-06-06T11:04:43.910035Z
	CreatedAt time.Time `json:"createdAt"`
	// example: 12345
	BytesOut uint64 `json:"bytesOut"`
	// example: 23451
	BytesIn uint64 `json:"bytesIn"`
	// example: 4cfb0324-daf6-4ad8-448b-e61fe0a1f918
	ServiceID string `json:"serviceId"`
	// example: 500000
	TokensEarned uint64 `json:"tokensEarned"`
}
