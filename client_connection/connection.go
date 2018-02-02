package client_connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/session"
)

type ConnectionStateChannel chan State

type connection struct {
	dialog         communication.Dialog
	vpnClient      openvpn.Client
	status         ConnectionStatus
	currentSession session.SessionID
	stateChannel   ConnectionStateChannel
}

func newConnection(dialog communication.Dialog, vpnClient openvpn.Client, currentSession session.SessionID, channel vpnStateChannel) *connection {

	conn := &connection{
		dialog,
		vpnClient,
		statusNotConnected(),
		currentSession,
		make(ConnectionStateChannel, 1),
	}

	go func() {
		for {
			conn.onVpnStateChanged(<-channel)
		}
	}()

	return conn
}

func (conn *connection) close() {
	conn.status = statusDisconnecting()
	conn.vpnClient.Stop()
}

func (conn *connection) onVpnStateChanged(state openvpn.State) {
	switch state {
	case openvpn.STATE_CONNECTING:
		conn.status = statusConnecting()
		conn.stateChannel <- Connecting
	case openvpn.STATE_CONNECTED:
		conn.status = statusConnected(conn.currentSession)
		conn.stateChannel <- Connected
	case openvpn.STATE_RECONNECTING:
		conn.status = statusReconnecting()
		conn.stateChannel <- Reconnecting
	case openvpn.STATE_EXITING:
		conn.dialog.Close()
		conn.status = statusNotConnected()
		conn.stateChannel <- NotConnected
		close(conn.stateChannel)
	}
}
