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

type Status struct {
	State     State
	SessionId string
}

type Manager interface {
	Connect(identity dto.Identity, NodeKey string) error
	Status() Status
	Disconnect() error
	Wait() error
}

type nullManager struct {
	errorChannel chan error
}

func NewManager() *nullManager {
	return &nullManager{make(chan error)}
}

func (nm *nullManager) Connect(identity dto.Identity, NodeKey string) error {
	return nil
}

func (nm *nullManager) Status() Status {
	return Status{NOT_CONNECTED, ""}
}

func (nm *nullManager) Disconnect() error {
	nm.errorChannel <- errors.New("disconnected")
	return nil
}

func (nm *nullManager) Wait() error {
	return <-nm.errorChannel
}
