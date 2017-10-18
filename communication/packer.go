package communication

type Packer interface {
	Pack() (data []byte)
}

type Unpacker interface {
	Unpack(data []byte)
}
