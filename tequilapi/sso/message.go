package sso

import (
	"encoding/json"
)

// MystnodesMessage expected by mystnodes.com
type MystnodesMessage struct {
	CodeChallenge string `json:"codeChallenge"`
	Identity      string `json:"identity"`
	RedirectURL   string `json:"redirectUrl"`
}

func (msg MystnodesMessage) json() ([]byte, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return []byte{}, err
	}
	return payload, nil
}
