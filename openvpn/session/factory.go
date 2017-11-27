package session

import (
	node_session "github.com/mysterium/node/session"
)

func NewVpnSession(manager node_session.ManagerInterface, config string) (session VpnSession) {
	id := manager.Create()

	session = VpnSession{
		Id:     id,
		Config: config,
	}

	return
}
