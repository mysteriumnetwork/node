package session

import (
	node_session "github.com/mysterium/node/session"
)

var manager node_session.Manager

func NewVpnSession(config string) (session VpnSession, err error) {
	id, err := manager.Create()
	if err != nil {
		return VpnSession{}, err
	}

	vpnSession := VpnSession{
		Id:     id,
		Config: config,
	}

	return vpnSession, nil
}
