package server

import (
	"errors"
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

	serviceLocation, err := detectCountry(cmd.ipResolver, cmd.locationDetector)
	if err != nil {
		return err
	}
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

	if err := cmd.mysteriumClient.RegisterProposal(proposal, signer); err != nil {
		return err
	}
	go func() {
		for {
			select {
			case <-time.After(1 * time.Minute):
				err := cmd.mysteriumClient.PingProposal(proposal, signer)
				if err != nil {
					log.Error("Failed to ping proposal", err)
					// do not stop server on missing ping to discovery. More on this in MYST-362 and MYST-370
				}
			case <-stopPinger:
				log.Info("Stopping proposal pinger")
				return
			}
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
