package client_connection

import (
	"errors"
	"github.com/mysterium/node/service_discovery/dto"
)

type fakeManager struct {
	errorChannel chan error
}

func NewFakeManager() *fakeManager {
	return &fakeManager{make(chan error)}
}

func (nm *fakeManager) Connect(identity dto.Identity, NodeKey string) error {
	return nil
}

func (nm *fakeManager) Status() ConnectionStatus {
	return ConnectionStatus{NOT_CONNECTED, ""}
}

func (nm *fakeManager) Disconnect() error {
	nm.errorChannel <- errors.New("disconnected")
	return nil
}

func (nm *fakeManager) Wait() error {
	return <-nm.errorChannel
}
