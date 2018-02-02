package client_connection

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
)

type vpnStateChannel chan openvpn.State

func channelToStateCallback(channel vpnStateChannel) state.ClientStateCallback {
	return func(state openvpn.State) {
		channel <- state
	}
}
