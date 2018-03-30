package dto

type NodeUnregisterRequest struct {
	// Unique identifier of a provider
	ProviderID string `json:"provider_id"`
}
