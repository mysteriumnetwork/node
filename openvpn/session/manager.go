package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

//NewManager returns session manager which maintans a map of session id -> session
func NewManager(clientConfig *openvpn.ClientConfig) *manager {
	return &manager{
		idGenerator:  &session.UUIDGenerator{},
		clientConfig: clientConfig,
		sessionMap:   make(map[session.SessionID]session.Session),
	}
}

type manager struct {
	idGenerator  session.Generator
	clientConfig *openvpn.ClientConfig
	sessionMap   map[session.SessionID]session.Session
}

func (manager *manager) Create(peerId identity.Identity) (sessionInstance session.Session, err error) {
	sessionInstance.ID = manager.idGenerator.Generate()
	sessionInstance.ConsumerIdentity = peerId
	sessionInstance.Config, err = openvpn.ConfigToString(*manager.clientConfig.Config)
	if err != nil {
		return
	}

	manager.sessionMap[sessionInstance.ID] = sessionInstance
	return sessionInstance, nil
}

func (manager *manager) FindSession(id session.SessionID) (session.Session, bool) {
	sessionInstance, found := manager.sessionMap[id]
	return sessionInstance, found
}
