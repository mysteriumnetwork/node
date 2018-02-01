package client_connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

type connection struct {
	dialog         communication.Dialog
	vpnClient      openvpn.Client
	status         ConnectionStatus
	currentSession session.SessionID
}

func newConnection(dialog communication.Dialog, vpnClient openvpn.Client, currentSession session.SessionID) *connection {
	return &connection{
		dialog,
		vpnClient,

		statusConnecting(),
		currentSession,
	}
}

func (conn *connection) onVpnStateChanged(state openvpn.State) {
	switch state {
	case openvpn.STATE_CONNECTED:
		conn.status = statusConnected(conn.currentSession)
	case openvpn.STATE_RECONNECTING:
		conn.status = statusConnecting()
	case openvpn.STATE_EXITING:
		conn.status = statusNotConnected()
	}
}

func (conn *connection) close() {

}
