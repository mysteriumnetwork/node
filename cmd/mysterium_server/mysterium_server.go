package main

import (
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
)

const SERVER_NODE_KEY = "12345"

func main() {
	ipifyClient := ipify.NewClient()
	vpnServerIp, err := ipifyClient.GetIp()
	if err != nil {
		panic(err)
	}

	vpnClientConfig := openvpn.NewClientConfig(
		vpnServerIp,
		"ca.crt", "client.crt", "client.key",
		"ta.key",
	)
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		panic(err)
	}

	mysterium := server.NewClient()
	if err := mysterium.NodeRegister(SERVER_NODE_KEY, vpnClientConfigString); err != nil {
		panic(err)
	}

	vpnServerConfig := openvpn.NewServerConfig(
		"10.8.0.0", "255.255.255.0",
		"ca.crt", "server.crt", "server.key",
		"dh.pem", "crl.pem", "ta.key",
	)
	vpnServer := openvpn.NewServer(vpnServerConfig)
	if err := vpnServer.Start(); err != nil {
		panic(err)
	}
	vpnServer.Wait()
}
