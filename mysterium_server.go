package main

import (
	"github.com/mysterium/node/openvpn"
)

const SERVER_NODE_KEY = "12345"

func main() {
	vpnConfig := openvpn.NewServerConfig()
	vpnServer := openvpn.NewServer(vpnConfig)
	vpnServer.Start()
}
