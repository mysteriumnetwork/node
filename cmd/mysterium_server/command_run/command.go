package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	vpn_session "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/server"
	dto_server "github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"io"
	"time"
	"errors"
	"github.com/mysterium/node/identity"
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
	providerId, err := selectIdentity(options.DirectoryKeystore, options.NodeKey)
	if err != nil {
		return err
	}

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

	proposal := service_discovery.NewServiceProposal(
		*providerId,
		nats.NewContact(*providerId),
	)

	sessionResponseHandler := vpn_session.CreateResponseHandler{
		ProposalId: proposal.Id,
		SessionManager: &session.Manager{
			Generator: &session.Generator{},
		},
		ClientConfigFactory: func() *openvpn.ClientConfig {
			return openvpn.NewClientConfig(
				vpnServerIp,
				options.DirectoryConfig+"/ca.crt",
				options.DirectoryConfig+"/client.crt",
				options.DirectoryConfig+"/client.key",
				options.DirectoryConfig+"/ta.key",
			)
		},
	}
	handleDialog := func(sender communication.Sender, receiver communication.Receiver) {
		receiver.Respond(communication.SESSION_CREATE, sessionResponseHandler.Handle)
	}

	cmd.communicationServer = cmd.CommunicationServerFactory(*providerId)
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

func (cmd *CommandRun) Wait() error {
	return cmd.vpnServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.vpnServer.Stop()
	cmd.communicationServer.Stop()
	cmd.NatService.Stop()
}

func selectIdentity(dir string, nodeKey string) (id *dto_discovery.Identity, err error) {
	identityController := identity.NewIdentityController(dir)

	// validate and return user provided identity
	if len(nodeKey) > 0 {
		id = identityController.GetIdentityByValue(nodeKey)
		if id == nil {
			return id, errors.New("identity doesn't exist")
		}
		identityController.CacheIdentity(id)
		return
	}

	// try cache
	id = identityController.GetIdentityFromCache()
	if id != nil {
		identityController.CacheIdentity(id)
		return
	}

	// if all fails, create a new one
	id, err = identityController.CreateIdentity()
	if err != nil {
		return id, err
	}

	identityController.CacheIdentity(id)

	return
}
