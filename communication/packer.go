package communication

type Packer func() (data []byte)
type Unpacker func(data []byte)
