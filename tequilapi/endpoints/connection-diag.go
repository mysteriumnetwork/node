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
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gvisor.dev/gvisor/pkg/sync"

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

	consumerAddress string

	db *gorm.DB
}

// NewConnectionDiagEndpoint creates and returns connection endpoint
func NewConnectionDiagEndpoint(manager connection.DiagManager, stateProvider stateProvider, proposalRepository proposalRepository, identityRegistry identityRegistry, publisher eventbus.Publisher, subscriber eventbus.Subscriber, addressProvider addressProvider, identitySelector selector.Handler) *ConnectionDiagEndpoint {
	ce := &ConnectionDiagEndpoint{
		manager:            manager,
		publisher:          publisher,
		subscriber:         subscriber,
		stateProvider:      stateProvider,
		proposalRepository: proposalRepository,
		identityRegistry:   identityRegistry,
		addressProvider:    addressProvider,
		identitySelector:   identitySelector,
	}

	chainID := config.GetInt64(config.FlagChainID)
	consumerID_, err := ce.identitySelector.UseOrCreate(config.FlagIdentity.Value, config.FlagIdentityPassphrase.Value, chainID)
	if err != nil {
		panic(err)
	}
	log.Error().Msgf("Unlocked identity: %v", consumerID_.Address)
	ce.consumerAddress = consumerID_.Address

	dsn := "host=____ user=mypguser password=___ dbname=myst_nodes port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	ce.db = db
	if err != nil {
		panic(err)
	}

	return ce
}

func dedupeSortedStrings(s []string) []string {
	if len(s) < 2 {
		return s
	}
	var e = 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			continue
		}
		s[e] = s[i]
		e++
	}

	return s[:e]
}

// DiagBatch is used to start a given providers check (batch mode)
func (ce *ConnectionDiagEndpoint) DiagBatch(c *gin.Context) {
	hermes, err := ce.addressProvider.GetActiveHermes(config.GetInt64(config.FlagChainID))
	if err != nil {
		c.Error(apierror.Internal("Failed to get active hermes", contract.ErrCodeActiveHermes))
		return
	}

	provs := make([]string, 0)
	c.Bind(&provs)
	sort.Strings(provs)
	provs = dedupeSortedStrings(provs)

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	resultMap := make(map[string]contract.ConnectionDiagInfoDTO, len(provs))
	wg.Add(len(provs))

	for _, prov := range provs {
		go func(prov string) {
			result := contract.ConnectionDiagInfoDTO{
				ProviderID: prov,
			}
			defer func() {
				mu.Lock()
				resultMap[prov] = result
				mu.Unlock()

				wg.Done()
			}()

			cr := &contract.ConnectionCreateRequest{
				ConsumerID:     ce.consumerAddress,
				ProviderID:     prov,
				Filter:         contract.ConnectionCreateFilter{IncludeMonitoringFailed: true},
				HermesID:       hermes.Hex(),
				ServiceType:    "wireguard",
				ConnectOptions: contract.ConnectOptions{},
			}
			if err := cr.Validate(); err != nil {
				result.Error = err.Error()
				return
			}

			consumerID := identity.FromAddress(cr.ConsumerID)
			status, err := ce.identityRegistry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), consumerID)
			if err != nil {
				log.Error().Err(err).Stack().Msg("Could not check registration status")
				result.Error = (contract.ErrCodeIDRegistrationCheck)
				return
			}
			switch status {
			case registry.Unregistered, registry.RegistrationError, registry.Unknown:
				log.Error().Msgf("Identity %q is not registered, aborting...", cr.ConsumerID)
				result.Error = (contract.ErrCodeIDNotRegistered)
				return
			case registry.InProgress:
				log.Info().Msgf("identity %q registration is in progress, continuing...", cr.ConsumerID)
			case registry.Registered:
				log.Info().Msgf("identity %q is registered, continuing...", cr.ConsumerID)
			default:
				log.Error().Msgf("identity %q has unknown status, aborting...", cr.ConsumerID)
				result.Error = (contract.ErrCodeIDStatusUnknown)
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

			if ce.manager.HasConnection(cr.ProviderID) {
				result.Error = (contract.ErrCodeConnectionAlreadyExists)
				return
			}

			err = ce.manager.Connect(consumerID, common.HexToAddress(cr.HermesID), proposalLookup, getConnectOptions(cr))
			if err != nil {
				switch err {
				case connection.ErrAlreadyExists:
					result.Error = (contract.ErrCodeConnectionAlreadyExists)
				case connection.ErrConnectionCancelled:
					result.Error = (contract.ErrCodeConnectionCancelled)
				default:
					log.Error().Err(err).Msgf("Failed to connect: %v", prov)
					result.Error = (contract.ErrCodeConnect)
				}
				return
			}

			resChannel := ce.manager.GetReadyChan(cr.ProviderID)
			res := <-resChannel
			log.Error().Msgf("Result > %v", res)

		}(prov)
	}
	wg.Wait()

	out := make([]contract.ConnectionDiagInfoDTO, 0)
	for _, prov := range provs {
		out = append(out, resultMap[prov])
	}
	utils.WriteAsJSON(out, c.Writer)
}

// Diag is used to start a given provider check
func (ce *ConnectionDiagEndpoint) Diag(c *gin.Context) {
	log.Debug().Msgf("Diag >>>")

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
		ConsumerID:     ce.consumerAddress,
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

	if ce.manager.HasConnection(cr.ProviderID) {
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
	}
	utils.WriteAsJSON(resp, c.Writer)
}

type proposalDB struct {
	ID        string
	Error     string
	DiagError string `json:"diag_error"`
	Country   string
}

func (proposalDB) TableName() string {
	return "node"
}

// DiagBatch is used to start a given providers check (batch mode)
func (ce *ConnectionDiagEndpoint) DiagBatch2(c *gin.Context) {

	hermes, err := ce.addressProvider.GetActiveHermes(config.GetInt64(config.FlagChainID))
	if err != nil {
		c.Error(apierror.Internal("Failed to get active hermes", contract.ErrCodeActiveHermes))
		return
	}

	country := c.Query("location")
	f := &proposal.Filter{
		ServiceType:             "wireguard",
		LocationCountry:         country,
		ExcludeUnsupported:      true,
		IncludeMonitoringFailed: true,
	}
	pp, err := ce.proposalRepository.Proposals(f)
	if err != nil {
		log.Error().Err(err).Stack().Msg("Proposals>")
	}
	log.Error().Msgf("pp> %v", len(pp))

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	resultMap := make(map[string]contract.ConnectionDiagInfoDTO, len(pp))
	wg.Add(len(pp))

	maxGoroutines := 15
	guard := make(chan struct{}, maxGoroutines)

	for _, pr := range pp {
		guard <- struct{}{}

		worker := func(provID string) (result contract.ConnectionDiagInfoDTO) {
			result.ProviderID = provID

			cr := &contract.ConnectionCreateRequest{
				ConsumerID:     ce.consumerAddress,
				ProviderID:     provID,
				Filter:         contract.ConnectionCreateFilter{IncludeMonitoringFailed: true},
				HermesID:       hermes.Hex(),
				ServiceType:    "wireguard",
				ConnectOptions: contract.ConnectOptions{},
			}
			if err := cr.Validate(); err != nil {
				result.Error = err.Error()
				return
			}

			consumerID := identity.FromAddress(cr.ConsumerID)
			status, err := ce.identityRegistry.GetRegistrationStatus(config.GetInt64(config.FlagChainID), consumerID)
			if err != nil {
				log.Error().Err(err).Stack().Msg("Could not check registration status")
				result.Error = (contract.ErrCodeIDRegistrationCheck)
				return
			}
			switch status {
			case registry.Unregistered, registry.RegistrationError, registry.Unknown:
				log.Error().Msgf("Identity %q is not registered, aborting...", cr.ConsumerID)
				result.Error = (contract.ErrCodeIDNotRegistered)
				return
			case registry.InProgress:
				log.Info().Msgf("identity %q registration is in progress, continuing...", cr.ConsumerID)
			case registry.Registered:
				log.Info().Msgf("identity %q is registered, continuing...", cr.ConsumerID)
			default:
				log.Error().Msgf("identity %q has unknown status, aborting...", cr.ConsumerID)
				result.Error = (contract.ErrCodeIDStatusUnknown)
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

			if ce.manager.HasConnection(cr.ProviderID) {
				result.Error = (contract.ErrCodeConnectionAlreadyExists)
				return
			}

			err = ce.manager.Connect(consumerID, common.HexToAddress(cr.HermesID), proposalLookup, getConnectOptions(cr))
			if err != nil {
				switch err {
				case connection.ErrAlreadyExists:
					result.Error = (contract.ErrCodeConnectionAlreadyExists)
				case connection.ErrConnectionCancelled:
					result.Error = (contract.ErrCodeConnectionCancelled)
				default:
					log.Error().Err(err).Msgf("Failed to connect: %v", provID)
					result.Error = (contract.ErrCodeConnect)
				}
				return
			}

			resChannel := ce.manager.GetReadyChan(cr.ProviderID)
			res := <-resChannel
			log.Error().Msgf("Result > %v", res)

			ev := res.(quality.DiagEvent)
			// result.Status = ev.Result
			if ev.Error != nil {
				result.DiagError = ev.Error.Error()
			}

			return
		}
		go func(pr proposal.PricedServiceProposal) {

			result := worker(pr.ProviderID)

			mu.Lock()
			resultMap[pr.ProviderID] = result
			mu.Unlock()

			// update
			provRec := proposalDB{ID: result.ProviderID, Country: pr.Location.Country}
			provRec.Error = ""
			provRec.DiagError = ""
			provRec.Error = result.Error
			provRec.DiagError = result.DiagError
			if ce.db.Model(&provRec).Select("Error", "DiagError", "Country").Updates(provRec).RowsAffected == 0 {
				ce.db.Create(&provRec)
			}

			wg.Done()
			<-guard
		}(pr)

	}
	wg.Wait()

	out := make([]contract.ConnectionDiagInfoDTO, 0)
	for _, prov := range pp {
		res := resultMap[prov.ProviderID]
		if res.Error != "" || res.DiagError != "" {
			out = append(out, resultMap[prov.ProviderID])
		}

	}
	utils.WriteAsJSON(out, c.Writer)
}

// AddRoutesForConnectionDiag adds proder check route to given router
func AddRoutesForConnectionDiag(
	manager connection.DiagManager,
	stateProvider stateProvider,
	proposalRepository proposalRepository,
	identityRegistry identityRegistry,
	publisher eventbus.Publisher,
	subscriber eventbus.Subscriber,
	addressProvider addressProvider,
	identitySelector selector.Handler,
	options node.Options,
) func(*gin.Engine) error {
	ConnectionDiagEndpoint := NewConnectionDiagEndpoint(manager, stateProvider, proposalRepository, identityRegistry, publisher, subscriber, addressProvider, identitySelector)
	return func(e *gin.Engine) error {
		connGroup := e.Group("")
		{
			connGroup.GET("/prov-checker", ConnectionDiagEndpoint.Diag)
			connGroup.POST("/prov-checker-batch", ConnectionDiagEndpoint.DiagBatch)
			connGroup.GET("/prov-checker-batch2", ConnectionDiagEndpoint.DiagBatch2)
		}
		return nil
	}
}
