package command_run

import (
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/ipify"
	"io"
)

type commandRun struct {
	output      io.Writer
	outputError io.Writer
}

func (cmd *commandRun) Run(args []string) error {
	options, err := cmd.parseArguments(args)
	if err != nil {
		return err
	}

	ipifyClient := ipify.NewClient()
	vpnServerIp, err := ipifyClient.GetIp()
	if err != nil {
		return err
	}

	vpnClientConfig := openvpn.NewClientConfig(
		vpnServerIp,
		options.DirectoryConfig + "/ca.crt",
		options.DirectoryConfig + "/client.crt",
		options.DirectoryConfig + "/client.key",
		options.DirectoryConfig + "/ta.key",
	)
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		return err
	}

	mysterium := server.NewClient()
	if err := mysterium.NodeRegister(options.NodeKey, vpnClientConfigString); err != nil {
		return err
	}

	vpnServerConfig := openvpn.NewServerConfig(
		"10.8.0.0", "255.255.255.0",
		options.DirectoryConfig + "/ca.crt",
		options.DirectoryConfig + "/server.crt",
		options.DirectoryConfig + "/server.key",
		options.DirectoryConfig + "/dh.pem",
		options.DirectoryConfig + "/crl.pem",
		options.DirectoryConfig + "/ta.key",
	)
	vpnServer := openvpn.NewServer(vpnServerConfig)
	if err := vpnServer.Start(); err != nil {
		return err
	}

	vpnServer.Wait()
	return nil
}