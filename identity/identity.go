package identity

import "strings"

type Identity struct {
	Address string `json:"address"`
}

func FromAddress(address string) Identity {
	return Identity{
		Address: strings.ToLower(address),
	}
}
