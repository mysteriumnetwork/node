package main

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
)

const NODE_KEY = "12345"

func main() {
	mysterium := server.NewClient()
	mysterium.SessionCreate(NODE_KEY)

	vpnConfig := openvpn.NewClientConfig("68.235.53.140", "pre-shared.key")
	vpnClient := openvpn.NewClient(vpnConfig)
	vpnClient.Start()
}
