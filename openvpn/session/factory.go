package session

import (
	node_session "github.com/mysterium/node/session"
)

var manager node_session.Manager

func NewVpnSession(config string) (session VpnSession) {
	id := manager.Create()

	session = VpnSession{
		Id:     id,
		Config: config,
	}

	return
}
