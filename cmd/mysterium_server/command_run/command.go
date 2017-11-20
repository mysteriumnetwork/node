package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/server"
	dto_server "github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"io"
	"time"
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

	handleDialog := func(sender communication.Sender, receiver communication.Receiver) {
		receiver.Respond(communication.GET_CONNECTION_CONFIG, func(request string) (response string) {
			return buildVpnClientConfig(vpnServerIp, options.DirectoryConfig)
		})
	}

	cmd.communicationServer = cmd.CommunicationServerFactory(providerId)
	if err = cmd.communicationServer.ServeDialogs(handleDialog); err != nil {
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

func buildVpnClientConfig(vpnIp string, dir string) (string) {
	vpnClientConfig := openvpn.NewClientConfig(
		vpnIp,
		dir+"/ca.crt",
		dir+"/client.crt",
		dir+"/client.key",
		dir+"/ta.key",
	)

	config, _ := openvpn.ConfigToString(*vpnClientConfig.Config)

	return config
}

func (cmd *CommandRun) Wait() error {
	return cmd.vpnServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.vpnServer.Stop()
	cmd.communicationServer.Stop()
	cmd.NatService.Stop()
}
