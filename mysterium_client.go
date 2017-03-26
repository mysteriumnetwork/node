package main

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
)

const CLIENT_NODE_KEY = "12345"

func main() {
	mysterium := server.NewClient()
	vpnSession, err := mysterium.SessionCreate(CLIENT_NODE_KEY)
	if err != nil {
		panic(err)
	}

	vpnConfig, err := openvpn.NewClientConfigFromString(vpnSession.ConnectionConfig)
	if err != nil {
		panic(err)
	}

	vpnClient := openvpn.NewClient(vpnConfig)
	if err := vpnClient.Start(); err != nil {
		panic(err)
	}

	vpnClient.Wait()
}
