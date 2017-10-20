package communication

type Packer interface {
	Pack() ([]byte, error)
}

type Unpacker interface {
	Unpack([]byte) error
}
