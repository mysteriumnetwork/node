package dto

// SessionStats mapped to json structure
type SessionStats struct {
	BytesSent       int    `json:"bytes_sent"`
	BytesReceived   int    `json:"bytes_received"`
	ProviderID      string `json:"provider_id"`
	ConsumerCountry string `json:"consumer_country"`
}
