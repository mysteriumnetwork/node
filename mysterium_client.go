package main

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
)

const CLIENT_NODE_KEY = "12345"

func main() {
	mysterium := server.NewClient()
	if _, err := mysterium.SessionCreate(CLIENT_NODE_KEY); err != nil {
		panic(err)
	}

	vpnConfig := openvpn.NewClientConfig("68.235.53.140", "pre-shared.key")
	vpnClient := openvpn.NewClient(vpnConfig)
	if err := vpnClient.Start(); err != nil {
		panic(err)
	}
}
