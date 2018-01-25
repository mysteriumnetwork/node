package command_run

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
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
	identityLoader   func() (identity.Identity, error)
	createSigner     identity.SignerFactory
	ipResolver       ip.Resolver
	mysteriumClient  server.Client
	natService       nat.NATService
	locationDetector location.Detector

	dialogWaiterFactory func(identity identity.Identity) (communication.DialogWaiter, dto_discovery.Contact)
	dialogWaiter        communication.DialogWaiter

	sessionManagerFactory func(serverIP string) session.ManagerInterface

	vpnServerFactory func() *openvpn.Server
	vpnServer        *openvpn.Server
}

func (cmd *CommandRun) Run() (err error) {
	providerID, err := cmd.identityLoader()
	if err != nil {
		return err
	}

	var providerContact dto_discovery.Contact
	cmd.dialogWaiter, providerContact = cmd.dialogWaiterFactory(providerID)

	// if for some reason we will need truly external IP, use GetPublicIP()
	vpnServerIP, err := cmd.ipResolver.GetOutboundIP()
	if err != nil {
		return err
	}

	cmd.natService.Add(nat.RuleForwarding{
		SourceAddress: "10.8.0.0/24",
		TargetIP:      vpnServerIP,
	})
	if err = cmd.natService.Start(); err != nil {
		return err
	}

	country, err := detectCountry(cmd.ipResolver, cmd.locationDetector)
	if err != nil {
		return err
	}
	log.Info("Country detected: ", country)

	location := dto_discovery.Location{Country: country}

	proposal := service_discovery.NewServiceProposalWithLocation(providerID, providerContact, location)

	sessionCreateConsumer := &session.SessionCreateConsumer{
		CurrentProposalID: proposal.ID,
		SessionManager:    cmd.sessionManagerFactory(vpnServerIP),
	}
	if err = cmd.dialogWaiter.ServeDialogs(sessionCreateConsumer); err != nil {
		return err
	}

	cmd.vpnServer = cmd.vpnServerFactory()
	if err := cmd.vpnServer.Start(); err != nil {
		return err
	}

	signer := cmd.createSigner(providerID)

	if err := cmd.mysteriumClient.RegisterProposal(proposal, signer); err != nil {
		return err
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			cmd.mysteriumClient.NodeSendStats(providerID.Address, signer)
		}
	}()

	return nil
}

func detectCountry(ipResolver ip.Resolver, locationDetector location.Detector) (string, error) {
	ip, err := ipResolver.GetPublicIP()
	if err != nil {
		return "", errors.New("IP detection failed: " + err.Error())
	}

	country, err := locationDetector.DetectCountry(ip)
	if err != nil {
		return "", errors.New("Country detection failed: " + err.Error())
	}
	return country, nil
}

func (cmd *CommandRun) Wait() error {
	return cmd.vpnServer.Wait()
}

func (cmd *CommandRun) Kill() error {
	cmd.vpnServer.Stop()
	err := cmd.dialogWaiter.Stop()
	if err != nil {
		return err
	}
	err = cmd.natService.Stop()

	return err
}
