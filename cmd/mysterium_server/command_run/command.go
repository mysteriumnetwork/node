package command_run

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ipify"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/service_discovery"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/pkg/errors"
	"time"
)

type CommandRun struct {
	identityLoader  func() (identity.Identity, error)
	createSigner    identity.SignerFactory
	ipifyClient     ipify.Client
	mysteriumClient server.Client
	natService      nat.NATService

	dialogWaiterFactory func(identity identity.Identity) (communication.DialogWaiter, dto_discovery.Contact)
	dialogWaiter        communication.DialogWaiter

	sessionManagerFactory func(serverIp string) session.ManagerInterface

	vpnServerFactory func() *openvpn.Server
	vpnServer        *openvpn.Server
}

func (cmd *CommandRun) Run() (err error) {
	providerId, err := cmd.identityLoader()
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
	if err = cmd.natService.Start(); err != nil {
		return err
	}

	country, err := detectCountry()
	if err != nil {
		return err
	}
	log.Info("Country detected: ", country)

	location := dto_discovery.Location{Country: country}
	proposal := service_discovery.NewServiceProposalWithLocation(providerId, providerContact, location)

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

	signer := cmd.createSigner(providerId)

	if err := cmd.mysteriumClient.RegisterProposal(proposal, signer); err != nil {
		return err
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			cmd.mysteriumClient.NodeSendStats(providerId.Address, signer)
		}
	}()

	return nil
}

func detectCountry() (string, error) {
	ipifyClient := ipify.NewClient()
	ip, err := ipifyClient.GetPublicIP()
	if err != nil {
		return "", errors.New("IP detection failed: " + err.Error())
	}

	country, err := location.DetectCountry(ip)
	if err != nil {
		return "", errors.New("Country detection failed: " + err.Error())
	}
	return country, nil
}

func (cmd *CommandRun) Wait() error {
	return cmd.vpnServer.Wait()
}

func (cmd *CommandRun) Kill() {
	cmd.vpnServer.Stop()
	cmd.dialogWaiter.Stop()
	cmd.natService.Stop()
}
