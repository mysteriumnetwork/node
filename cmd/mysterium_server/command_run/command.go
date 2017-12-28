package command_run

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/server"
	dto_server "github.com/mysterium/node/server/dto"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"time"
)

type CommandRun struct {
	identitySelector func() (identity.Identity, error)
	ipifyClient      ipify.Client
	mysteriumClient  server.Client
	natService       nat.NATService

	dialogWaiterFactory func(identity identity.Identity) (communication.DialogWaiter, dto_discovery.Contact)
	dialogWaiter        communication.DialogWaiter

	sessionManagerFactory func(serverIp string) session.ManagerInterface

	vpnServerFactory func() *openvpn.Server
	vpnServer        *openvpn.Server
}

func (cmd *CommandRun) Run() (err error) {
	providerId, err := cmd.identitySelector()
	if err != nil {
		return err
	}

	var providerContact dto_discovery.Contact
	cmd.dialogWaiter, providerContact = cmd.dialogWaiterFactory(providerId)

	// if for some reason we will need truly external IP, use GetPublicIP()
	vpnServerIp, err := cmd.ipifyClient.GetOutboundIP()
	if err != nil {
		return err
	}

	cmd.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIp:      vpnServerIp,
	})

	// clear probable stale entries
	cmd.natService.Stop()

	if err = cmd.natService.Start(); err != nil {
		return err
	}

	proposal := service_discovery.NewServiceProposal(providerId, providerContact)

	sessionCreateConsumer := &session.SessionCreateConsumer{
		CurrentProposalId: proposal.Id,
		SessionManager:    cmd.sessionManagerFactory(vpnServerIp),
	}
	if err = cmd.dialogWaiter.ServeDialogs(sessionCreateConsumer); err != nil {
		return err
	}

	cmd.vpnServer = cmd.vpnServerFactory()
	if err := cmd.vpnServer.Start(); err != nil {
		return err
	}

	if err := cmd.mysteriumClient.NodeRegister(proposal); err != nil {
		return err
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			cmd.mysteriumClient.NodeSendStats(providerId.Address, []dto_server.SessionStatsDeprecated{})
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
	cmd.natService.Stop()
}
