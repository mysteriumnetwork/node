package command_run

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/ipify"
)

const SERVER_NODE_KEY = "12345"

type commandRun struct {}

// NewCommand return a new instance of commandRun.
func NewCommandRun() *commandRun {
	return &commandRun{}
}

func (cmd *commandRun) Run(args ...string) error {
	ipifyClient := ipify.NewClient()
	vpnServerIp, err := ipifyClient.GetIp()
	if err != nil {
		return err
	}

	vpnClientConfig := openvpn.NewClientConfig(
		vpnServerIp,
		"ca.crt", "client.crt", "client.key",
		"ta.key",
	)
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		return err
	}

	mysterium := server.NewClient()
	if err := mysterium.NodeRegister(SERVER_NODE_KEY, vpnClientConfigString); err != nil {
		return err
	}

	vpnServerConfig := openvpn.NewServerConfig(
		"10.8.0.0", "255.255.255.0",
		"ca.crt", "server.crt", "server.key",
		"dh.pem", "crl.pem", "ta.key",
	)
	vpnServer := openvpn.NewServer(vpnServerConfig)
	if err := vpnServer.Start(); err != nil {
		return err
	}

	vpnServer.Wait()
	return nil
}