package dto

type NodeStatsRequest struct {
	NodeKey  	string         `json:"node_key"`
	Sessions 	[]SessionStats `json:"sessions"`
	NodeVersion string 			`json:"node_version"`
}
