package client

import "fmt"

// StatusDTO holds connection status and session id
type StatusDTO struct {
	Status    string `json:"status"`
	SessionID string `json:"sessionId"`
}

// StatisticsDTO holds statistics about connection
type StatisticsDTO struct {
	BytesSent     int `json:"bytesSent"`
	BytesReceived int `json:"bytesReceived"`
	Duration      int `json:"duration"`
}

// ProposalList describes list of proposals
type ProposalList struct {
	Proposals []ProposalDTO `json:"proposals"`
}

// ProposalDTO describes service proposal
type ProposalDTO struct {
	ID                int                  `json:"id"`
	ProviderID        string               `json:"providerId"`
	ServiceDefinition ServiceDefinitionDTO `json:"serviceDefinition"`
}

func (p ProposalDTO) String() string {
	return fmt.Sprintf("Id: %d , Provider: %s, Country: %s", p.ID, p.ProviderID, p.ServiceDefinition.LocationOriginate.Country)
}

// ServiceDefinitionDTO describes service of proposal
type ServiceDefinitionDTO struct {
	LocationOriginate LocationDTO `json:"locationOriginate"`
}

// LocationDTO describes location
type LocationDTO struct {
	Country string `json:"country"`
}

// IdentityDTO holds identity address
type IdentityDTO struct {
	Address string `json:"id"`
}

// IdentityList holds returned list of identities
type IdentityList struct {
	Identities []IdentityDTO `json:"identities"`
}
