package identity

type Identity struct {
	Address string
}

func FromAddress(address string) Identity {
	return Identity{address}
}
