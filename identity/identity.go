package identity

type Identity struct {
	Id string
}

func NewIdentity(id string) Identity {
	return Identity{id}
}
