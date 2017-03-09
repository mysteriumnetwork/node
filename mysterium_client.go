package main

import (
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
)

const SERVER_HOST = "server.mysterium.localhost"
const CLIENT_NODE_KEY = "12345"

func main() {
	mysterium := server.NewClient()
	if _, err := mysterium.SessionCreate(CLIENT_NODE_KEY); err != nil {
		panic(err)
	}

	vpnConfig := openvpn.NewClientConfig(SERVER_HOST, "pre-shared.key")
	vpnClient := openvpn.NewClient(vpnConfig)
	if err := vpnClient.Start(); err != nil {
		panic(err)
	}

	vpnClient.Wait()
}
