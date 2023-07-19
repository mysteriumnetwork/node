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

package mysterium

import (
	"encoding/json"
	"strings"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/services"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/services/datatransfer"
	"github.com/mysteriumnetwork/node/services/dvpn"
	"github.com/mysteriumnetwork/node/services/scraping"
	"github.com/mysteriumnetwork/node/services/wireguard"
)

// DefaultProviderNodeOptions returns default options.
func DefaultProviderNodeOptions() *MobileNodeOptions {
	options := DefaultNodeOptionsByNetwork(string(config.Mainnet))
	options.IsProvider = true
	options.TequilapiSecured = true
	return options
}

func (mb *MobileNode) unlockIdentity(adr, passphrase string) string {
	chainID := config.GetInt64(config.FlagChainID)
	id, err := mb.identitySelector.UseOrCreate(adr, passphrase, chainID)
	if err != nil {
		return ""
	}
	return id.Address
}

// StartProvider starts all provider services (provider mode)
func (mb *MobileNode) StartProvider() {
	providerID := mb.unlockIdentity(
		config.FlagIdentity.Value,
		config.FlagIdentityPassphrase.Value,
	)
	log.Info().Msgf("Unlocked identity: %v", providerID)

	serviceTypes := strings.Split(config.Current.GetString(config.FlagActiveServices.Name), ",")
	if len(serviceTypes) == 1 && serviceTypes[0] == "" {
		serviceTypes = []string{}
	}
	stoppedServices := strings.Split(config.Current.GetString(config.FlagStoppedServices.Name), ",")
	if len(stoppedServices) == 1 && stoppedServices[0] == "" {
		stoppedServices = []string{}
	}

	if len(serviceTypes) == 0 {
		if len(stoppedServices) > 0 {

			// restore stopped service
			serviceTypes = stoppedServices
			stoppedServices = []string{}
		} else {

			// on the first run enable data scraping service
			if config.Current.GetString(contract.TermsProviderAgreed) == "" {
				serviceTypes = []string{scraping.ServiceType}
			}
		}
	}

	setUserConfigRaw(config.FlagStoppedServices.Name, strings.Join(stoppedServices, ","))
	setUserConfigRaw(config.FlagActiveServices.Name, strings.Join(serviceTypes, ","))
	config.Current.SaveUserConfig()

	// set of services to be started
	serviceTypesSet := make(map[string]bool)
	for _, serviceType := range serviceTypes {
		serviceTypesSet[serviceType] = true
	}
	for _, srv := range mb.servicesManager.List(true) {
		if srv.State() == servicestate.Running {
			delete(serviceTypesSet, srv.Type)
		}
	}

	for serviceType := range serviceTypesSet {
		serviceOpts, err := services.GetStartOptions(serviceType)
		if err != nil {
			log.Error().Err(err).Msg("GetStartOptions failed")
			return
		}

		_, err = mb.servicesManager.Start(identity.Identity{Address: providerID}, serviceType, serviceOpts.AccessPolicyList, serviceOpts.TypeOptions)
		if err != nil {
			log.Error().Err(err).Msg("servicesManager.Start failed")
			return
		}
	}
}

// StopProvider stops all provider services, started by StartProvider
func (mb *MobileNode) StopProvider() {
	haveActive := false
	for _, srv := range mb.servicesManager.List(true) {
		if srv.State() == servicestate.Running {
			haveActive = true
			break
		}
	}

	stoppedServices := strings.Split(config.Current.GetString(config.FlagStoppedServices.Name), ",")
	if len(stoppedServices) == 1 && stoppedServices[0] == "" {
		stoppedServices = []string{}
	}

	if haveActive {
		stoppedServices = []string{}
		for _, srv := range mb.servicesManager.List(true) {
			if srv.State() != servicestate.Running {
				continue
			}

			stoppedServices = append(stoppedServices, srv.Type)
			err := mb.servicesManager.Stop(srv.ID)
			if err != nil {
				log.Error().Err(err).Msg("servicesManager.Stop failed")
				return
			}
		}
	}

	setUserConfigRaw(config.FlagStoppedServices.Name, strings.Join(stoppedServices, ","))
	setUserConfigRaw(config.FlagActiveServices.Name, "")
	config.Current.SaveUserConfig()
}

// SetFlagLauncherVersion sets LauncherVersion flag value, which is reported to Prometheus
func SetFlagLauncherVersion(val string) {
	config.Current.SetDefault(config.FlagLauncherVersion.Name, val)
}

func getAllServiceTypes() []string {
	return []string{wireguard.ServiceType, scraping.ServiceType, datatransfer.ServiceType, dvpn.ServiceType}
}

// GetServiceTypes returns all possible service types
func GetServiceTypes() ([]byte, error) {
	result := getAllServiceTypes()
	return json.Marshal(&result)
}

// ServicesState - state of service
type ServicesState struct {
	Service string `json:"id"`
	State   string `json:"state"`
}

// GetAllServicesState returns state of all services
func (mb *MobileNode) GetAllServicesState() ([]byte, error) {
	result := make([]ServicesState, 0)
	for _, srv := range mb.servicesManager.List(true) {
		result = append(result, ServicesState{srv.Type, string(srv.State())})
	}
	return json.Marshal(&result)
}
