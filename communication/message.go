package communication

type MessageType string

type MessagePacker func() (data []byte)

type MessageUnpacker func(data []byte)
