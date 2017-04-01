package dto

type NodeRegisterRequest struct {
	NodeKey          string `json:"node_key"`
	ConnectionConfig string `json:"connection_config"`
}
