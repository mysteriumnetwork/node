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

package node

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
)

func MockSessionProvider(monitoringFailed bool) ProviderSessions {
	return func(providerID string) []Session {
		return []Session{
			{
				ProviderID:       providerID,
				ServiceType:      "wireguard",
				MonitoringFailed: monitoringFailed,
			},
		}
	}
}

type mockCurrentIdentity struct {
	identityLocked bool
	identity       string
}

func newMockCurrentIdentity(identity string, identityLocked bool) *mockCurrentIdentity {
	return &mockCurrentIdentity{
		identity:       identity,
		identityLocked: identityLocked,
	}
}

func (mci *mockCurrentIdentity) GetUnlockedIdentity() (identity.Identity, bool) {
	if mci.identityLocked {
		return identity.Identity{}, false
	}

	return identity.Identity{Address: mci.identity}, true
}

func TestShit(t *testing.T) {

	for _, data := range []struct {
		monitoringFailed bool
		identityLocked   bool
		identity         string
		expectedStatus   MonitoringStatus
	}{
		{
			monitoringFailed: true,
			identityLocked:   false,
			identity:         "0xa",
			expectedStatus:   Failed,
		},
		{
			monitoringFailed: false,
			identityLocked:   false,
			identity:         "0xa",
			expectedStatus:   Passed,
		},
		{
			monitoringFailed: false,
			identityLocked:   true,
			identity:         "0xa",
			expectedStatus:   Pending,
		},
		{
			monitoringFailed: true,
			identityLocked:   true,
			identity:         "0xa",
			expectedStatus:   Pending,
		},
	} {
		t.Run(fmt.Sprintf("status %s when %+v", data.expectedStatus, data), func(t *testing.T) {
			fixture := NewMonitoringStatusTracker(
				MockSessionProvider(data.monitoringFailed),
				newMockCurrentIdentity(data.identity, data.identityLocked),
			)

			assert.Equal(t, data.expectedStatus, fixture.Status())
		})
	}
}
