package session

import "github.com/mysterium/node/identity"

// ManagerFake represents fake manager usually useful in tests
type ManagerFake struct{}

var fakeVpnConfig = VPNConfig{
	RemoteIP:        "1.2.3.4",
	RemotePort:      1234,
	RemoteProtocol:  "tcp",
	TLSPresharedKey: "preshared-key",
	CACertificate:   "ca-certificate",
}

// Create function creates and returns fake session
func (manager *ManagerFake) Create(peerID identity.Identity) (Session, error) {
	return Session{"new-id", fakeVpnConfig, peerID}, nil
}

// FindSession always returns empty session and signals that session is not found
func (manager *ManagerFake) FindSession(id SessionID) (Session, bool) {
	return Session{}, false
}
