package identity

import "strings"

// Identity represents unique user network identity
type Identity struct {
	Address string `json:"address"`
}

// FromAddress converts address to identity
func FromAddress(address string) Identity {
	return Identity{
		Address: strings.ToLower(address),
	}
}
