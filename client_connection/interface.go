package client_connection

import (
	"errors"
	"github.com/mysterium/node/service_discovery/dto"
)

type State string

const (
	NOT_CONNECTED = State("NOT_CONNECTED")
	NEGOTIATING   = State("NEGOTIATING")
	CONNECTED     = State("CONNECTED")
)

var (
	ALREADY_CONNECTED = errors.New("already connected")
)

type ConnectionStatus struct {
	State     State
	SessionId string
}

type Manager interface {
	Connect(identity dto.Identity, NodeKey string) error
	Status() ConnectionStatus
	Disconnect() error
	Wait() error
}
