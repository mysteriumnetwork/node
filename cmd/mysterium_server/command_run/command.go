package command_run

import (
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	"io"
)

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	ipifyClient ipify.Client
	mysteriumClient server.Client
	vpnServer *openvpn.Server
}

func (cmd *commandRun) Run(options CommandOptions) error {
	vpnServerIp, err := cmd.ipifyClient.GetIp()
	if err != nil {
		return err
	}

	vpnClientConfig := openvpn.NewClientConfig(
		vpnServerIp,
		options.DirectoryConfig+"/ca.crt",
		options.DirectoryConfig+"/client.crt",
		options.DirectoryConfig+"/client.key",
		options.DirectoryConfig+"/ta.key",
	)
	vpnClientConfigString, err := openvpn.ConfigToString(*vpnClientConfig.Config)
	if err != nil {
		return err
	}

	if err := cmd.mysteriumClient.NodeRegister(options.NodeKey, vpnClientConfigString); err != nil {
		return err
	}

	vpnServerConfig := openvpn.NewServerConfig(
		"10.8.0.0", "255.255.255.0",
		options.DirectoryConfig+"/ca.crt",
		options.DirectoryConfig+"/server.crt",
		options.DirectoryConfig+"/server.key",
		options.DirectoryConfig+"/dh.pem",
		options.DirectoryConfig+"/crl.pem",
		options.DirectoryConfig+"/ta.key",
	)
	cmd.vpnServer = openvpn.NewServer(vpnServerConfig, options.DirectoryRuntime)
	if err := cmd.vpnServer.Start(); err != nil {
		return err
	}

	return nil
}

func (cmd *commandRun) Wait() {
	cmd.vpnServer.Wait()
}

func (cmd *commandRun) Kill() error {
	return cmd.vpnServer.Stop()
}