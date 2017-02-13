package main

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
)

const SERVER_NODE_KEY = "12345"

func main() {
	mysterium := server.NewClient()
	if err := mysterium.NodeRegister(SERVER_NODE_KEY); err != nil {
		panic(err)
	}

	vpnConfig := openvpn.NewServerConfig("pre-shared.key")
	vpnServer := openvpn.NewServer(vpnConfig)
	if err := vpnServer.Start(); err != nil {
		panic(err)
	}
}
