package client_connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
)

type connection struct {
	dialog         communication.Dialog
	vpnClient      openvpn.Client
	status         ConnectionStatus
	currentSession session.SessionID
}

type vpnClientCreator func(callback state.ClientStateCallback) (*openvpn.Client, error)

func newConnection(dialog communication.Dialog, currentSession session.SessionID, createVpnClient vpnClientCreator) (*connection , error) {

	vpnClient, err := createVpnClient(conn.)

	return &connection{
		dialog,
		vpnClient,

		statusConnecting(),
		currentSession,
	},nil
}

func (conn *connection) onVpnStateChanged(connection ,state openvpn.State) {
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
