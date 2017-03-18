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

	vpnConfig := openvpn.NewServerConfig(
		"10.8.0.0", "255.255.255.0",
		"ca.crt", "server.crt", "server.key",
		"dh.pem", "crl.pem", "ta.key",
	)
	vpnServer := openvpn.NewServer(vpnConfig)
	if err := vpnServer.Start(); err != nil {
		panic(err)
	}

	vpnServer.Wait()
}
