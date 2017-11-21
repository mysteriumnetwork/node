package session

import (
	sid "github.com/mysterium/node/session"
)

func NewVpnSession(config string) (session VpnSession, err error) {
	id, err := sid.GenerateSessionId()
	if err != nil {
		return VpnSession{}, err
	}

	vpnSession := VpnSession{
		Id:     id,
		Config: config,
	}

	return vpnSession, nil
}
