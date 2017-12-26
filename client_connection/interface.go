package client_connection

import (
	"github.com/mysterium/node/identity"
)

type State string

const (
	NotConnected = State("Not Connected")
	Connecting   = State("Connecting")
	Connected    = State("Connected")
)

type ConnectionStatus struct {
	State     State
	SessionId string
	LastError error
}

type Manager interface {
	Connect(identity identity.Identity, NodeKey string) error
	Status() ConnectionStatus
	Disconnect() error
	Wait() error
}
