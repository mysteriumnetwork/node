/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package service

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/location/locationstate"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/core/policy/localcopy"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/p2p"
	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	"github.com/mysteriumnetwork/node/services/scraping"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/mysteriumnetwork/node/session/connectivity"
	"github.com/mysteriumnetwork/node/utils/netutil"
	"github.com/mysteriumnetwork/node/utils/reftracker"
)

var (
	// ErrorLocation error indicates that action (i.e. disconnect)
	ErrorLocation = errors.New("failed to detect service location")
	// ErrUnsupportedServiceType indicates that manager tried to create an unsupported service type
	ErrUnsupportedServiceType = errors.New("unsupported service type")
	// ErrUnsupportedAccessPolicy indicates that manager tried to create service with unsupported access policy
	ErrUnsupportedAccessPolicy = errors.New("unsupported access policy")
)

const (
	channelIdleTimeout = 1 * time.Minute
)

// Service interface represents pluggable Mysterium service
type Service interface {
	Serve(instance *Instance) error
	Stop() error
	ConfigProvider
}

// DiscoveryFactory initiates instance which is able announce service discoverability
type DiscoveryFactory func() Discovery

// Discovery registers the service to the discovery api periodically
type Discovery interface {
	Start(ownIdentity identity.Identity, proposal func() market.ServiceProposal)
	Stop()
	Wait()
}

// LocationResolver detects location for service proposal.
type locationResolver interface {
	DetectLocation() (locationstate.Location, error)
}

// WaitForNATHole blocks until NAT hole is punched towards consumer through local NAT or until hole punching failed
type WaitForNATHole func() error

// NewManager creates new instance of pluggable instances manager
func NewManager(
	serviceRegistry *Registry,
	discoveryFactory DiscoveryFactory,
	eventPublisher Publisher,
	policyOracle *localcopy.Oracle,
	policyProvider policy.Provider,
	p2pListener p2p.Listener,
	sessionManager func(service *Instance, channel p2p.Channel) *SessionManager,
	statusStorage connectivity.StatusStorage,
	location locationResolver,
) *Manager {
	return &Manager{
		serviceRegistry:  serviceRegistry,
		servicePool:      NewPool(eventPublisher),
		discoveryFactory: discoveryFactory,
		eventPublisher:   eventPublisher,
		policyOracle:     policyOracle,
		policyProvider:   policyProvider,
		p2pListener:      p2pListener,
		sessionManager:   sessionManager,
		statusStorage:    statusStorage,
		location:         location,
	}
}

// Manager entrypoint which knows how to start pluggable Mysterium instances
type Manager struct {
	serviceRegistry *Registry
	servicePool     *Pool

	discoveryFactory DiscoveryFactory
	eventPublisher   Publisher
	policyOracle     *localcopy.Oracle
	policyProvider   policy.Provider

	p2pListener    p2p.Listener
	sessionManager func(service *Instance, channel p2p.Channel) *SessionManager
	statusStorage  connectivity.StatusStorage
	location       locationResolver
}

// Start starts an instance of the given service type if knows one in service registry.
// It passes the options to the start method of the service.
// If an error occurs in the underlying service, the error is then returned.
func (manager *Manager) Start(providerID identity.Identity, serviceType string, policyIDs []string, options Options) (id ID, err error) {
	log.Debug().Fields(map[string]interface{}{
		"providerID":  providerID.Address,
		"serviceType": serviceType,
		"policyIDs":   policyIDs,
		"options":     options,
	}).Msg("Starting service")
	service, err := manager.serviceRegistry.Create(serviceType, options)
	if err != nil {
		return id, err
	}

	accessPolicies := manager.policyOracle.Policies(policyIDs)
	var policyProvider policy.Provider
	if len(policyIDs) == 1 && policyIDs[0] == "mysterium" {
		policyProvider = manager.policyProvider
	} else {
		policyRules := localcopy.NewRepository()
		if len(policyIDs) > 0 {
			if err = manager.policyOracle.SubscribePolicies(accessPolicies, policyRules); err != nil {
				log.Warn().Err(err).Msg("Can't find given access policyOracle")
				return id, ErrUnsupportedAccessPolicy
			}
		}
		policyProvider = policyRules
	}

	location, err := manager.location.DetectLocation()
	if err != nil {
		return "", err
	}

	proposal := market.NewProposal(providerID.Address, serviceType, market.NewProposalOpts{
		Location:       market.NewLocation(location),
		AccessPolicies: accessPolicies,
		Contacts:       []market.Contact{manager.p2pListener.GetContact()},
	})

	discovery := manager.discoveryFactory()

	id, err = generateID()
	if err != nil {
		return id, err
	}

	instance := &Instance{
		ID:             id,
		ProviderID:     providerID,
		Type:           serviceType,
		state:          servicestate.Starting,
		Options:        options,
		service:        service,
		Proposal:       proposal,
		policyProvider: policyProvider,
		discovery:      discovery,
		eventPublisher: manager.eventPublisher,
		location:       manager.location,
	}

	discovery.Start(providerID, instance.proposalWithCurrentLocation)

	channelHandlers := func(ch p2p.Channel) {
		chID := "channel:" + ch.ID()
		log.Info().Msgf("tracking p2p.Channel: %q", chID)
		reftracker.Singleton().Put(chID, channelIdleTimeout, func() {
			log.Debug().Msgf("collecting unused p2p.Channel %q", chID)
			ch.Close()
		})
		instance.addP2PChannel(ch)
		mng := manager.sessionManager(instance, ch)
		subscribeSessionCreate(mng, ch)
		subscribeSessionStatus(ch, manager.statusStorage)
		subscribeSessionAcknowledge(mng, ch)
		subscribeSessionDestroy(mng, ch)
		subscribeSessionPayments(mng, ch)
	}
	stopP2PListener, err := manager.p2pListener.Listen(providerID, serviceType, channelHandlers)
	if err != nil {
		return id, fmt.Errorf("could not subscribe to p2p channels: %w", err)
	}

	manager.servicePool.Add(instance)

	go func() {
		instance.setState(servicestate.Running)

		serveErr := service.Serve(instance)
		if serveErr != nil {
			log.Error().Err(serveErr).Msg("Service serve failed")
		}

		stopP2PListener()

		stopErr := manager.servicePool.Stop(id)
		if stopErr != nil {
			log.Error().Err(stopErr).Msg("Service stop failed")
		}

		discovery.Wait()
	}()

	netutil.LogNetworkStats()

	return id, nil
}

func generateID() (ID, error) {
	uid, err := uuid.NewV4()
	if err != nil {
		return ID(""), err
	}
	return ID(uid.String()), nil
}

// List returns array of service instances.
func (manager *Manager) List(includeAll bool) []*Instance {
	runningInstances := manager.servicePool.List()
	if !includeAll {
		return runningInstances
	}

	added := map[string]bool{
		wireguard.ServiceType:    false,
		scraping.ServiceType:     false,
		datatransfer.ServiceType: false,
		dvpn.ServiceType:         false,
	}

	result := make([]*Instance, 0, len(added))
	for _, instance := range runningInstances {
		result = append(result, instance)
		added[instance.Type] = true
	}

	for serviceType, alreadyAdded := range added {
		if alreadyAdded {
			continue
		}

		result = append(result, &Instance{
			Type:  serviceType,
			state: servicestate.NotRunning,
		})
	}

	return result
}

// Kill stops all services.
func (manager *Manager) Kill() error {
	return manager.servicePool.StopAll()
}

// Stop stops the service.
func (manager *Manager) Stop(id ID) error {
	err := manager.servicePool.Stop(id)
	if err != nil {
		return err
	}

	return nil
}

// Service returns a service instance by requested id.
func (manager *Manager) Service(id ID) *Instance {
	return manager.servicePool.Instance(id)
}
