package dto

type SessionStats struct {
	Id            string `json:"session_key"`
	BytesSent     int    `json:"bytes_sent"`
	BytesReceived int    `json:"bytes_received"`
}
