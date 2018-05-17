package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)


var mockedVPNConfig = "config_string"

type mockedConfigProvider func() string

func (mcp mockedConfigProvider) ProvideServiceConfig() (session.ServiceConfiguration, error) {
	return mcp(), nil
}

func provideMockedVPNConfig() string {
	return mockedVPNConfig
}

type MockIdentityExtractor struct {
	OnExtractReturnIdentity identity.Identity
	OnExtractReturnError    error
}

type MockSessionManager struct {
	OnFindReturnSession session.Session
	OnFindReturnSuccess bool
}


func (manager *MockSessionManager) Create(peerID identity.Identity) (sessionInstance session.Session, err error) {
	return session.Session{}, nil
}

func (manager *MockSessionManager) FindSession(sessionId session.SessionID) (session.Session, bool) {
	return manager.OnFindReturnSession, manager.OnFindReturnSuccess
}

func (extractor *MockIdentityExtractor) Extract(message []byte, signature identity.Signature) (identity.Identity, error) {
	return extractor.OnExtractReturnIdentity, extractor.OnExtractReturnError
}
