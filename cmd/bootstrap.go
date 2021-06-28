// +build !ios,!android

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

	"github.com/mysteriumnetwork/node/ui"
	uinoop "github.com/mysteriumnetwork/node/ui/noop"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/tequilapi"
	tequilapi_endpoints "github.com/mysteriumnetwork/node/tequilapi/endpoints"
	"github.com/mysteriumnetwork/node/utils"
)

func (di *Dependencies) bootstrapTequilapi(nodeOptions node.Options, listener net.Listener) (tequilapi.APIServer, error) {
	if !nodeOptions.TequilapiEnabled {
		return tequilapi.NewNoopAPIServer(), nil
	}

	router := tequilapi.NewAPIRouter()
	tequilapi_endpoints.AddRoutesForDocs(router)
	tequilapi_endpoints.AddRouteForStop(router, utils.SoftKiller(di.Shutdown))
	tequilapi_endpoints.AddRoutesForAuthentication(router, di.Authenticator, di.JWTAuthenticator)
	tequilapi_endpoints.AddRoutesForIdentities(router, di.IdentityManager, di.IdentitySelector, di.IdentityRegistry, di.ConsumerBalanceTracker, di.AddressProvider, di.HermesChannelRepository, di.BCHelper, di.Transactor, di.BeneficiaryProvider, di.IdentityMover)
	tequilapi_endpoints.AddRoutesForConnection(router, di.ConnectionManager, di.StateKeeper, di.ProposalRepository, di.IdentityRegistry, di.EventBus, di.AddressProvider)
	tequilapi_endpoints.AddRoutesForSessions(router, di.SessionStorage)
	tequilapi_endpoints.AddRoutesForConnectionLocation(router, di.IPResolver, di.LocationResolver, di.LocationResolver)
	tequilapi_endpoints.AddRoutesForProposals(router, di.ProposalRepository, di.PricingHelper, di.LocationResolver, di.FilterPresetStorage)
	tequilapi_endpoints.AddRoutesForService(router, di.ServicesManager, services.JSONParsersByType, di.ProposalRepository)
	tequilapi_endpoints.AddRoutesForAccessPolicies(di.HTTPClient, router, config.GetString(config.FlagAccessPolicyAddress))
	tequilapi_endpoints.AddRoutesForNAT(router, di.StateKeeper)
	tequilapi_endpoints.AddRoutesForTransactor(router, di.IdentityRegistry, di.Transactor, di.HermesPromiseSettler, di.SettlementHistoryStorage, di.AddressProvider, di.BeneficiarySaver, di.BeneficiaryProvider)
	tequilapi_endpoints.AddRoutesForConfig(router)
	tequilapi_endpoints.AddRoutesForMMN(router, di.MMN)
	tequilapi_endpoints.AddRoutesForFeedback(router, di.Reporter)
	tequilapi_endpoints.AddRoutesForConnectivityStatus(router, di.SessionConnectivityStatusStorage)
	tequilapi_endpoints.AddRoutesForCurrencyExchange(router, di.PilvytisAPI)
	tequilapi_endpoints.AddRoutesForPilvytis(router, di.PilvytisAPI)
	tequilapi_endpoints.AddRoutesForTerms(router)
	if err := tequilapi_endpoints.AddRoutesForSSE(router, di.StateKeeper, di.EventBus); err != nil {
		return nil, err
	}

	if config.GetBool(config.FlagPProfEnable) {
		tequilapi_endpoints.AddRoutesForPProf(router)
	}

	corsPolicy := tequilapi.NewMysteriumCorsPolicy()
	return tequilapi.NewServer(listener, router, corsPolicy), nil
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
	di.UIServer = ui.NewServer(bindAddress, options.UI.UIPort, options.TequilapiAddress, options.TequilapiPort, di.JWTAuthenticator, di.HTTPClient)
	return nil
}
