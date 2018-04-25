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
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/server"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
	"github.com/mysterium/node/version"
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

	vpnServerFactory func(sessionManager session.Manager, serviceLocation dto_discovery.Location,
		providerID identity.Identity, callback state.Callback) *openvpn.Server

	vpnServer    *openvpn.Server
	checkOpenvpn func() error
	protocol	string
}

// Start starts server - does not block
func (cmd *Command) Start() (err error) {
	log.Info("[Server version]", version.AsString())
	err = cmd.checkOpenvpn()
	if err != nil {
		return err
	}

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

	serviceCountry, err := cmd.locationDetector.DetectCountry()
	if err != nil {
		return err
	}
	log.Info("Country detected: ", serviceCountry)
	serviceLocation := dto_discovery.Location{Country: serviceCountry}

	proposal := discovery.NewServiceProposalWithLocation(providerID, providerContact, serviceLocation, cmd.protocol)

	sessionManager := cmd.sessionManagerFactory(vpnServerIP)

	dialogHandler := session.NewDialogHandler(proposal.ID, sessionManager)
	if err := cmd.dialogWaiter.ServeDialogs(dialogHandler); err != nil {
		return err
	}

	stopPinger := make(chan int)
	vpnStateCallback := func(state openvpn.State) {
		switch state {
		case openvpn.ConnectedState:
			log.Info("Open vpn service started")
		case openvpn.ExitingState:
			log.Info("Open vpn service exiting")
			close(stopPinger)
		}
	}
	cmd.vpnServer = cmd.vpnServerFactory(sessionManager, serviceLocation, providerID, vpnStateCallback)
	if err := cmd.vpnServer.Start(); err != nil {
		return err
	}

	signer := cmd.createSigner(providerID)

	for {
		err := cmd.mysteriumClient.RegisterProposal(proposal, signer)
		if err != nil {
			log.Errorf("Failed to register proposal: %v, retrying after 1 min.", err)
			time.Sleep(1 * time.Minute)
		} else {
			break
		}
	}

	go PingProposalLoop(proposal, cmd.mysteriumClient, signer, stopPinger)

	return nil
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

// Ping proposal until stopped, then unregister proposal
func PingProposalLoop(proposal dto_discovery.ServiceProposal, mysteriumClient  server.Client, signer identity.Signer, stopPinger <- chan int) {
	for {
		select {
		case <-time.After(1 * time.Minute):
			err := mysteriumClient.PingProposal(proposal, signer)
			if err != nil {
				log.Error("Failed to ping proposal", err)
				// do not stop server on missing ping to discovery. More on this in MYST-362 and MYST-370
			}
		case <-stopPinger:
			log.Info("Stopping proposal pinger")
			err := mysteriumClient.UnregisterProposal(proposal.ProviderID, signer)
			if err != nil {
				log.Error("Failed to unregister proposal", err)
			}
			return
		}
	}
}
