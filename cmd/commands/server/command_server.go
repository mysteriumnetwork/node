package server

import (
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/ip"
	"github.com/mysterium/node/location"
	"github.com/mysterium/node/nat"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/discovery"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/pkg/errors"
	"time"
)

// Command represent entrypoint for Mysterium server with top level components
type Command struct {
	identityLoader   func() (identity.Identity, error)
	createSigner     identity.SignerFactory
	ipResolver       ip.Resolver
	mysteriumClient  server.Client
	natService       nat.NATService
	locationDetector location.Detector

	dialogWaiterFactory func(identity identity.Identity) communication.DialogWaiter
	dialogWaiter        communication.DialogWaiter

	sessionManagerFactory func(serverIP string) session.Manager

	vpnServerFactory func(sessionManager session.Manager) *openvpn.Server
	vpnServer        *openvpn.Server
}

// Start starts server - does not block
func (cmd *Command) Start() (err error) {
	providerID, err := cmd.identityLoader()
	if err != nil {
		return err
	}

	cmd.dialogWaiter = cmd.dialogWaiterFactory(providerID)
	providerContact, err := cmd.dialogWaiter.Start()

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

	serviceLocation, err := detectCountry(cmd.ipResolver, cmd.locationDetector)
	if err != nil {
		return err
	}
	proposal := discovery.NewServiceProposalWithLocation(providerID, providerContact, serviceLocation)

	sessionManager := cmd.sessionManagerFactory(vpnServerIP)

	dialogHandler := session.NewDialogHandler(proposal.ID, sessionManager)
	if err := cmd.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return err
	}

	cmd.vpnServer = cmd.vpnServerFactory(sessionManager)
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
			cmd.mysteriumClient.PingProposal(proposal, signer)
		}
	}()

	return nil
}

func detectCountry(ipResolver ip.Resolver, locationDetector location.Detector) (dto_discovery.Location, error) {
	myIP, err := ipResolver.GetPublicIP()
	if err != nil {
		return dto_discovery.Location{}, errors.New("IP detection failed: " + err.Error())
	}

	myCountry, err := locationDetector.DetectCountry(myIP)
	if err != nil {
		return dto_discovery.Location{}, errors.New("Country detection failed: " + err.Error())
	}

	log.Info("Country detected: ", myCountry)
	return dto_discovery.Location{Country: myCountry}, nil
}

// Wait blocks until server is stopped
func (cmd *Command) Wait() error {
	return cmd.vpnServer.Wait()
}

// Kill stops server
func (cmd *Command) Kill() error {
	cmd.vpnServer.Stop()
	err := cmd.dialogWaiter.Stop()
	if err != nil {
		return err
	}
	err = cmd.natService.Stop()

	return err
}
