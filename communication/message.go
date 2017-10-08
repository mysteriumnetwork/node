package communication

type MessageType string

type MessageHandler func(message []byte)
