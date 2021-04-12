/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package identity

import (
	"github.com/mysteriumnetwork/node/core/location/locationstate"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/eventbus"
)

// AppTopicResidentCountry resident country event topic
const AppTopicResidentCountry = "resident-country"

type locationProvider interface {
	GetOrigin() locationstate.Location
}

// ResidentCountry for saving and loading resident country
// resident country is used by legal to pay VAT
type ResidentCountry struct {
	eventBus         eventbus.EventBus
	locationResolver locationProvider
}

// NewResidentCountry constructor
func NewResidentCountry(eventBus eventbus.EventBus, locationResolver locationProvider) *ResidentCountry {
	return &ResidentCountry{eventBus: eventBus, locationResolver: locationResolver}
}

// Save country code and fire AppTopicResidentCountry
func (rc *ResidentCountry) Save(identity, countryCode string) error {
	if len(countryCode) == 0 || len(identity) == 0 {
		return errors.New("identity and countryCode are required")
	}
	config.Current.SetUser(config.FlagResidentCountry.Name, countryCode)
	if err := config.Current.SaveUserConfig(); err != nil {
		return err
	}
	rc.publishResidentCountry(identity)
	return nil
}

// Get get stored resident country
func (rc *ResidentCountry) Get() string {
	stored := config.Current.GetString(config.FlagResidentCountry.Name)
	if len(stored) == 0 {
		return rc.locationResolver.GetOrigin().Country
	}
	return stored
}

func (rc *ResidentCountry) publishResidentCountry(identity string) {
	country := rc.Get()
	event := ResidentCountryEvent{
		ID:      identity,
		Country: country,
	}
	rc.eventBus.Publish(AppTopicResidentCountry, event)
}
