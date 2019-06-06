package event

import (
	"time"

	"github.com/mysteriumnetwork/node/market"
)

// Topic is the topic that we use to announce state changes to via the event bus
const Topic = "State change"

// State represents the node state at the current moment. It's a read only object, used only to display data.
type State struct {
	NATStatus NATStatus        `json:"natStatus"`
	Services  []ServiceInfo    `json:"serviceInfo"`
	Sessions  []ServiceSession `json:"sessions"`
}

// NATStatus stores the nat status related information
type NATStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ServiceInfo stores the information about a service
type ServiceInfo struct {
	ID             string                 `json:"id"`
	ProviderID     string                 `json:"providerId"`
	Type           string                 `json:"type"`
	Options        interface{}            `json:"options"`
	Status         string                 `json:"status"`
	Proposal       market.ServiceProposal `json:"proposal"`
	AccessPolicies *[]market.AccessPolicy `json:"accessPolicies,omitempty"`
	Sessions       []ServiceSession       `json:"serviceSession,omitempty"`
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
}
