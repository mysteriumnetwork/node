package identity

type Identity struct {
	Address string `json:"address"`
}

func FromAddress(address string) Identity {
	return Identity{address}
}
