package event

import "github.com/mysteriumnetwork/node/market"

const Topic = "State change"

type State struct {
	NATStatus NATStatus        `json:"natStatus"`
	Services  []ServiceInfo    `json:"serviceInfo"`
	Sessions  []ServiceSession `json:"sessions"`
}

type NATStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// ServiceInfo
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
	ID         string `json:"id"`
	ConsumerID string `json:"consumerId"`
}
