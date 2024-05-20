//go:build !ios && !android

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

package cmd

import (
	"net"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/consumer/entertainment"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_client "github.com/mysteriumnetwork/node/tequilapi/client"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/ui"
	uinoop "github.com/mysteriumnetwork/node/ui/noop"
	"github.com/mysteriumnetwork/node/ui/versionmanager"
	"github.com/mysteriumnetwork/node/utils"
)

func (di *Dependencies) bootstrapTequilapi(nodeOptions node.Options, listener net.Listener) (tequilapi.APIServer, error) {
	if !nodeOptions.TequilapiEnabled {
		return tequilapi.NewNoopAPIServer(), nil
	}
	tequilaApiClient := tequilapi_client.NewClient(nodeOptions.TequilapiAddress, nodeOptions.TequilapiPort)

	return tequilapi.NewServer(
		listener,
		nodeOptions,
		di.JWTAuthenticator,
		[]func(engine *gin.Engine) error{
			func(e *gin.Engine) error {
				if err := tequilapi_endpoints.AddRoutesForSSE(e, di.StateKeeper, di.EventBus); err != nil {
					return err
				}
				return nil
			},
			func(e *gin.Engine) error {
				if config.GetBool(config.FlagPProfEnable) {
					tequilapi_endpoints.AddRoutesForPProf(e)
				}
				return nil
			},
			func(e *gin.Engine) error {
				e.GET("/healthcheck", tequilapi_endpoints.HealthCheckEndpointFactory(time.Now, os.Getpid).HealthCheck)
				return nil
			},
			tequilapi_endpoints.AddRouteForStop(utils.SoftKiller(di.Shutdown)),
			tequilapi_endpoints.AddRoutesForAuthentication(di.Authenticator, di.JWTAuthenticator, di.SSOMystnodes),
			tequilapi_endpoints.AddRoutesForIdentities(di.IdentityManager, di.IdentitySelector, di.IdentityRegistry, di.ConsumerBalanceTracker, di.AddressProvider, di.HermesChannelRepository, di.BCHelper, di.Transactor, di.BeneficiaryProvider, di.IdentityMover, di.BeneficiaryAddressStorage, di.HermesMigrator),
			tequilapi_endpoints.AddRoutesForConnection(di.MultiConnectionManager, di.StateKeeper, di.ProposalRepository, di.IdentityRegistry, di.EventBus, di.AddressProvider),
			tequilapi_endpoints.AddRoutesForConnectionDiag(di.MultiConnectionDiagManager, di.StateKeeper, di.ProposalRepository, di.IdentityRegistry, di.EventBus, di.EventBus, di.AddressProvider, di.IdentitySelector, nodeOptions),
			tequilapi_endpoints.AddRoutesForSessions(di.SessionStorage),
			tequilapi_endpoints.AddRoutesForConnectionLocation(di.IPResolver, di.LocationResolver, di.LocationResolver),
			tequilapi_endpoints.AddRoutesForProposals(di.ProposalRepository, di.PricingHelper, di.LocationResolver, di.FilterPresetStorage, di.NATProber),
			tequilapi_endpoints.AddRoutesForService(di.ServicesManager, services.JSONParsersByType, di.ProposalRepository, tequilaApiClient),
			tequilapi_endpoints.AddRoutesForAccessPolicies(di.HTTPClient, config.GetString(config.FlagAccessPolicyAddress)),
			tequilapi_endpoints.AddRoutesForNAT(di.StateKeeper, di.NATProber),
			tequilapi_endpoints.AddRoutesForNodeUI(versionmanager.NewVersionManager(di.UIServer, di.HTTPClient, di.uiVersionConfig)),
			tequilapi_endpoints.AddRoutesForNode(di.NodeStatusTracker, di.NodeStatsTracker),
			tequilapi_endpoints.AddRoutesForTransactor(di.IdentityRegistry, di.Transactor, di.Affiliator, di.HermesPromiseSettler, di.SettlementHistoryStorage, di.AddressProvider, di.BeneficiaryProvider, di.BeneficiarySaver, di.PilvytisAPI),
			tequilapi_endpoints.AddRoutesForAffiliator(di.Affiliator),
			tequilapi_endpoints.AddRoutesForConfig,
			tequilapi_endpoints.AddRoutesForMMN(di.MMN, di.SSOMystnodes, di.Authenticator),
			tequilapi_endpoints.AddRoutesForFeedback(di.Reporter),
			tequilapi_endpoints.AddRoutesForConnectivityStatus(di.SessionConnectivityStatusStorage),
			tequilapi_endpoints.AddRoutesForDocs,
			tequilapi_endpoints.AddRoutesForCurrencyExchange(di.PilvytisAPI),
			tequilapi_endpoints.AddRoutesForPilvytis(di.PilvytisAPI, di.PilvytisOrderIssuer, di.LocationResolver),
			tequilapi_endpoints.AddRoutesForTerms,
			tequilapi_endpoints.AddEntertainmentRoutes(entertainment.NewEstimator(
				config.FlagPaymentPriceGiB.Value,
				config.FlagPaymentPriceHour.Value,
			)),
			tequilapi_endpoints.AddRoutesForValidator,
		},
	)
}

func (di *Dependencies) bootstrapNodeUIVersionConfig(nodeOptions node.Options) error {
	if !nodeOptions.TequilapiEnabled || nodeOptions.Directories.NodeUI == "" {
		noopCfg, _ := versionmanager.NewNoOpVersionConfig()
		di.uiVersionConfig = noopCfg
		return nil
	}

	versionConfig, err := versionmanager.NewVersionConfig(nodeOptions.Directories.NodeUI)
	if err != nil {
		return err
	}
	di.uiVersionConfig = versionConfig
	return nil
}

func (di *Dependencies) bootstrapUIServer(options node.Options) (err error) {
	if !options.UI.UIEnabled {
		di.UIServer = uinoop.NewServer()
		return nil
	}

	bindAddress := options.UI.UIBindAddress
	if bindAddress == "" {
		bindAddress, err = di.IPResolver.GetOutboundIP()
		if err != nil {
			return err
		}
		bindAddress = bindAddress + ",127.0.0.1"
	}
	di.UIServer = ui.NewServer(bindAddress, options.UI.UIPort, options.TequilapiAddress, options.TequilapiPort, di.JWTAuthenticator, di.HTTPClient, di.uiVersionConfig)
	return nil
}
