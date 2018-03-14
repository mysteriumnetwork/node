package dto

type SessionStats struct {
	BytesSent     int    `json:"bytes_sent"`
	BytesReceived int    `json:"bytes_received"`
	ProviderID    string `json:"provider_id"`
	ClientCountry string `json:"client_country"`
}
