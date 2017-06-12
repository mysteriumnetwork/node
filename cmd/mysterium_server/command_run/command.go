package command_run

import (
	"github.com/MysteriumNetwork/node/ipify"
	"github.com/MysteriumNetwork/node/nat"
	"github.com/MysteriumNetwork/node/openvpn"
	"github.com/MysteriumNetwork/node/server"
	"github.com/MysteriumNetwork/node/server/dto"
	"io"
	"time"
)

type commandRun struct {
	output      io.Writer
	outputError io.Writer

	ipifyClient     ipify.Client
	mysteriumClient server.Client
	natService      nat.NATService
	vpnServer       *openvpn.Server
}

func (cmd *commandRun) Run(options CommandOptions) error {
	vpnServerIp, err := cmd.ipifyClient.GetIp()
	if err != nil {
		return err
	}

	cmd.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIp:      vpnServerIp,
	})
	if err = cmd.natService.Start(); err != nil {
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

	if err := cmd.mysteriumClient.NodeRegister(options.NodeKey, vpnClientConfigString); err != nil {
		return err
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			cmd.mysteriumClient.NodeSendStats(options.NodeKey, []dto.SessionStats{})
		}
	}()

	return nil
}

func (cmd *commandRun) Wait() error {
	return cmd.vpnServer.Wait()
}

func (cmd *commandRun) Kill() {
	cmd.vpnServer.Stop()
	cmd.natService.Stop()
}
