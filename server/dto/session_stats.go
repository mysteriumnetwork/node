package dto

type SessionStats struct {
	BytesSent     int `json:"bytes_sent"`
	BytesReceived int `json:"bytes_received"`
}

// TODO: remove this struct in favor of `SessionStats`
type SessionStatsDeprecated struct {
	Id            string `json:"session_key"`
	BytesSent     int    `json:"bytes_sent"`
	BytesReceived int    `json:"bytes_received"`
}
