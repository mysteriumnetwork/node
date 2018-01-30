package client

// StatusDTO holds connection status and session id
type StatusDTO struct {
	Status    string `json:"status"`
	SessionID string `json:"sessionId"`
}

// StatisticsDTO holds statistics about connection
type StatisticsDTO struct {
	BytesSent       int `json:"bytesSent"`
	BytesReceived   int `json:"bytesReceived"`
	DurationSeconds int `json:"durationSeconds"`
}

// IdentityDTO holds identity address
type IdentityDTO struct {
	Address string `json:"id"`
}

// IdentityList holds returned list of identities
type IdentityList struct {
	Identities []IdentityDTO `json:"identities"`
}
