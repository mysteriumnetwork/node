package session

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

func NewManager(clientConfigFactory func() *openvpn.ClientConfig) *manager {
	return &manager{
		generator:           &session.Generator{},
		clientConfigFactory: clientConfigFactory,
		sessions:            make([]session.SessionId, 0),
	}
}

type manager struct {
	generator           session.GeneratorInterface
	clientConfigFactory func() *openvpn.ClientConfig
	sessions            []session.SessionId
}

func (manager *manager) Create() (sessionInstance session.Session, err error) {
	sessionInstance.Id = manager.generator.Generate()

	vpnClientConfig := manager.clientConfigFactory()
	sessionInstance.Config, err = openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		return
	}

	manager.Add(sessionInstance)
	return sessionInstance, nil
}

func (manager *manager) Add(session session.Session) {
	manager.sessions = append(manager.sessions, session.Id)
}
