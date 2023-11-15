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
	"os"
	"sync"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/config"

	"github.com/mysteriumnetwork/node/core/location/locationstate"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestResidentEvent(t *testing.T) {
	// given
	configFileName := NewTempFileName(t)
	defer os.Remove(configFileName)
	err := config.Current.LoadUserConfig(configFileName)

	bus := eventbus.New()
	residentCountry := NewResidentCountry(bus, newMockLocationResolver("UK"))

	atomic := &AtomicResidentCountryEvent{}
	err = bus.Subscribe(AppTopicResidentCountry, func(e ResidentCountryEvent) {
		atomic.Store(e)
	})
	assert.NoError(t, err)

	//when
	err = residentCountry.Save("0x123", "LT")

	//then
	assert.NoError(t, err)
	eventually(t, func() bool {
		return atomic.Load().ID == "0x123"
	})
	eventually(t, func() bool {
		return atomic.Load().Country == "LT"
	})
}

func NewTempFileName(t *testing.T) string {
	file, err := os.CreateTemp("", "*")
	assert.NoError(t, err)
	return file.Name()
}

type mockLocationResolver struct{ country string }

func newMockLocationResolver(country string) locationProvider {
	return &mockLocationResolver{country: country}
}

func (mlr *mockLocationResolver) GetOrigin() locationstate.Location {
	return locationstate.Location{Country: "LT"}
}

type AtomicResidentCountryEvent struct {
	lock  sync.Mutex
	value ResidentCountryEvent
}

func (a *AtomicResidentCountryEvent) Store(value ResidentCountryEvent) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.value = value
}

func (a *AtomicResidentCountryEvent) Load() ResidentCountryEvent {
	a.lock.Lock()
	defer a.lock.Unlock()
	return a.value
}

func eventually(t *testing.T, condition func() bool) {
	assert.Eventually(t, condition, 5*time.Second, 500*time.Millisecond)
}
