/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
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

package endpoints

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/mysteriumnetwork/node/tequilapi/utils"
)

// ConnectionDiagEndpoint struct represents /connection resource and it's subresources
type ConnectionDiagEndpoint struct {
	manager    connection.DiagManager
	publisher  eventbus.Publisher
	subscriber eventbus.Subscriber

	stateProvider stateProvider
	// TODO connection should use concrete proposal from connection params and avoid going to marketplace
	proposalRepository proposalRepository
	identityRegistry   identityRegistry
	addressProvider    addressProvider
	identitySelector   selector.Handler
}

// NewConnectionDiagEndpoint creates and returns connection endpoint
func NewConnectionDiagEndpoint(manager connection.DiagManager, stateProvider stateProvider, proposalRepository proposalRepository, identityRegistry identityRegistry, publisher eventbus.Publisher, subscriber eventbus.Subscriber, addressProvider addressProvider, identitySelector selector.Handler) *ConnectionDiagEndpoint {
	return &ConnectionDiagEndpoint{
		manager:            manager,
		publisher:          publisher,
		subscriber:         subscriber,
		stateProvider:      stateProvider,
		proposalRepository: proposalRepository,
		identityRegistry:   identityRegistry,
		addressProvider:    addressProvider,
		identitySelector:   identitySelector,
	}
}

// Diag is used to start provider check
func (ce *ConnectionDiagEndpoint) Diag(c *gin.Context) {
	log.Debug().Msgf("Diag >>>")

	chainID := config.GetInt64(config.FlagChainID)
	consumerID_, err := ce.identitySelector.UseOrCreate(config.FlagIdentity.Value, config.FlagIdentityPassphrase.Value, chainID)
	if err != nil {
		c.Error(apierror.Internal("Failed to unlock identity", err.Error()))
		return
	}
	log.Error().Msgf("Unlocked identity: %v", consumerID_)

	hermes, err := ce.addressProvider.GetActiveHermes(config.GetInt64(config.FlagChainID))
	if err != nil {
		c.Error(apierror.Internal("Failed to get active hermes", contract.ErrCodeActiveHermes))
		return
	}

	prov := c.Query("id")
	if len(prov) == 0 {
		c.Error(errors.New("Empty prameter: prov"))
		return
	}
	cr := &contract.ConnectionCreateRequest{
		ConsumerID:     consumerID_.Address,
		ProviderID:     prov,
		Filter:         contract.ConnectionCreateFilter{IncludeMonitoringFailed: true},
		HermesID:       hermes.Hex(),
		ServiceType:    "wireguard",
		ConnectOptions: contract.ConnectOptions{},
	}

	if err := cr.Validate(); err != nil {
		c.Error(err)
		return
	}

	consumerID := identity.FromAddress(cr.ConsumerID)
	status, err := ce.identityRegistry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), consumerID)
	if err != nil {
		log.Error().Err(err).Stack().Msg("Could not check registration status")
		c.Error(apierror.Internal("Failed to check ID registration status: "+err.Error(), contract.ErrCodeIDRegistrationCheck))
		return
	}

	switch status {
	case registry.Unregistered, registry.RegistrationError, registry.Unknown:
		log.Error().Msgf("Identity %q is not registered, aborting...", cr.ConsumerID)
		c.Error(apierror.Unprocessable(fmt.Sprintf("Identity %q is not registered. Please register the identity first", cr.ConsumerID), contract.ErrCodeIDNotRegistered))
		return
	case registry.InProgress:
		log.Info().Msgf("identity %q registration is in progress, continuing...", cr.ConsumerID)
	case registry.Registered:
		log.Info().Msgf("identity %q is registered, continuing...", cr.ConsumerID)
	default:
		log.Error().Msgf("identity %q has unknown status, aborting...", cr.ConsumerID)
		c.Error(apierror.Unprocessable(fmt.Sprintf("Identity %q has unknown status. Aborting", cr.ConsumerID), contract.ErrCodeIDStatusUnknown))
		return
	}

	if len(cr.ProviderID) > 0 {
		cr.Filter.Providers = append(cr.Filter.Providers, cr.ProviderID)
	}

	f := &proposal.Filter{
		ServiceType:             cr.ServiceType,
		LocationCountry:         cr.Filter.CountryCode,
		ProviderIDs:             cr.Filter.Providers,
		IPType:                  cr.Filter.IPType,
		IncludeMonitoringFailed: cr.Filter.IncludeMonitoringFailed,
		AccessPolicy:            "all",
	}
	proposalLookup := connection.FilteredProposals(f, cr.Filter.SortBy, ce.proposalRepository)

	hasConnection := ce.manager.HasConnection(cr.ProviderID)
	if hasConnection {
		c.Error(apierror.Unprocessable("Connection already exists", contract.ErrCodeConnectionAlreadyExists))
		return
	}

	err = ce.manager.Connect(consumerID, common.HexToAddress(cr.HermesID), proposalLookup, getConnectOptions(cr))
	if err != nil {
		switch err {
		case connection.ErrAlreadyExists:
			c.Error(apierror.Unprocessable("Connection already exists", contract.ErrCodeConnectionAlreadyExists))
		case connection.ErrConnectionCancelled:
			c.Error(apierror.Unprocessable("Connection cancelled", contract.ErrCodeConnectionCancelled))
		default:
			log.Error().Err(err).Msg("Failed to connect")
			c.Error(apierror.Internal("Failed to connect: "+err.Error(), contract.ErrCodeConnect))
		}
		return
	}

	resChannel := ce.manager.GetReadyChan(cr.ProviderID)
	res := <-resChannel
	log.Error().Msgf("Result > %v", res)

	resp := contract.ConnectionDiagInfoDTO{
		ProviderID: prov,
		Status:     res.(quality.DiagEvent).Result,
	}
	utils.WriteAsJSON(resp, c.Writer)
}

// AddRoutesForConnectionDiag adds proder check route to given router
func AddRoutesForConnectionDiag(
	manager connection.DiagManager,
	stateProvider stateProvider,
	proposalRepository proposalRepository,
	identityRegistry identityRegistry,
	publisher eventbus.Publisher,
	publisher2 eventbus.Subscriber,
	addressProvider addressProvider,
	identitySelector selector.Handler,
	options node.Options,
) func(*gin.Engine) error {
	ConnectionDiagEndpoint := NewConnectionDiagEndpoint(manager, stateProvider, proposalRepository, identityRegistry, publisher, publisher2, addressProvider, identitySelector)
	return func(e *gin.Engine) error {
		connGroup := e.Group("")
		{
			if options.ProvChecker {
				connGroup.GET("/prov-checker", ConnectionDiagEndpoint.Diag)
			}
		}
		return nil
	}
}
