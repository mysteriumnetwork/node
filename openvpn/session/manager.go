package session

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

//NewManager returns session manager which maintans a map of session id -> session
func NewManager(clientConfig *openvpn.ClientConfig) *manager {
	return &manager{
		idGenerator:  &session.Generator{},
		clientConfig: clientConfig,
		sessionMap:   make(map[session.SessionID]session.Session),
	}
}

type manager struct {
	idGenerator  session.GeneratorInterface
	clientConfig *openvpn.ClientConfig
	sessionMap   map[session.SessionID]session.Session
}

func (manager *manager) Create() (sessionInstance session.Session, err error) {
	sessionInstance.ID = manager.idGenerator.Generate()

	sessionInstance.Config, err = openvpn.ConfigToString(*manager.clientConfig.Config)
	if err != nil {
		return
	}

	manager.add(sessionInstance)
	return sessionInstance, nil
}

func (manager *manager) add(session session.Session) {
	manager.sessionMap[session.ID] = session
}

func (manager *manager) FindSession(id session.SessionID) (session.Session, bool) {
	sessionInstance, found := manager.sessionMap[id]
	return sessionInstance, found
}
