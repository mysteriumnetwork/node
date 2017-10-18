package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/server"
	dto_server "github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"io"
	"time"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/communication/nats"
)

type CommandRun struct {
	Output      io.Writer
	OutputError io.Writer

	IpifyClient     ipify.Client
	MysteriumClient server.Client
	NatService      nat.NATService

	CommunicationServerFactory func(identity dto_discovery.Identity) communication.Server
	communicationServer        communication.Server

	vpnMiddlewares []openvpn.ManagementMiddleware
	vpnServer      *openvpn.Server
}

func (cmd *CommandRun) Run(options CommandOptions) (err error) {
	providerId := dto_discovery.Identity(options.NodeKey)

	vpnServerIp, err := cmd.IpifyClient.GetIp()
	if err != nil {
		return err
	}

	cmd.NatService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIp:      vpnServerIp,
	})
	if err = cmd.NatService.Start(); err != nil {
		return err
	}

	cmd.communicationServer = cmd.CommunicationServerFactory(providerId)
	if err = cmd.communicationServer.ServeDialogs(cmd.handleDialog); err != nil {
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

	proposal := service_discovery.NewServiceProposal(
		providerId,
		nats.NewContact(providerId),
	)

	proposal.ConnectionConfig = vpnClientConfigString

	if err := cmd.MysteriumClient.NodeRegister(proposal); err != nil {
		return err
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			cmd.MysteriumClient.NodeSendStats(options.NodeKey, []dto_server.SessionStats{})
		}
	}()

	return nil
}

func (cmd *CommandRun) handleDialog(sender communication.Sender, receiver communication.Receiver) {

}

func (cmd *CommandRun) Wait() error {
	return cmd.vpnServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.vpnServer.Stop()
	cmd.communicationServer.Stop()
	cmd.NatService.Stop()
}
