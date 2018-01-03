package dto

type SessionStats struct {
	BytesSent     int `json:"bytes_sent"`
	BytesReceived int `json:"bytes_received"`
}
