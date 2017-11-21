package session

import "github.com/mysterium/node/session"

type VpnSession struct {
	Id     session.SessionId `json:"id"`
	Config string            `json:"config"`
}
