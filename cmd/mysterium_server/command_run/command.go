package command_run

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
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
)

type CommandRun struct {
	Output      io.Writer
	OutputError io.Writer

	IpifyClient     ipify.Client
	MysteriumClient server.Client
	NatService      nat.NATService

	DialogWaiterFactory func(identity identity.Identity) (communication.DialogWaiter, dto_discovery.Contact)
	dialogWaiter        communication.DialogWaiter

	SessionManager session.ManagerInterface

	vpnMiddlewares []openvpn.ManagementMiddleware
	vpnServer      *openvpn.Server
}

func (cmd *CommandRun) Run(options CommandOptions) (err error) {
	ks := keystore.NewKeyStore(options.DirectoryKeystore, keystore.StandardScryptN, keystore.StandardScryptP)
	identityHandler := identity.NewNodeIdentityHandler(ks, options.DirectoryKeystore)

	providerId, err := identityHandler.Select(options.NodeKey)
	if err != nil {
		providerId, err = identityHandler.Create()
		if err != nil {
			return err
		}

		if err := cmd.MysteriumClient.RegisterIdentity(providerId); err != nil {
			return err
		}
	}

	var providerContact dto_discovery.Contact
	cmd.dialogWaiter, providerContact = cmd.DialogWaiterFactory(providerId)

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

	proposal := service_discovery.NewServiceProposal(providerId, providerContact)

	sessionCreateConsumer := &vpn_session.SessionCreateConsumer{
		CurrentProposalId: proposal.Id,
		SessionManager:    cmd.SessionManager,
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
	if err = cmd.dialogWaiter.ServeDialogs(sessionCreateConsumer); err != nil {
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
	cmd.dialogWaiter.Stop()
	cmd.NatService.Stop()
}
