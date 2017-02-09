package dto

type Session struct {
	Id               string `json:"session_key"`
	ConnectionConfig string `json:"connection_config"`
}
