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

package monitoring

import (
	"fmt"
	"testing"

	"github.com/mysteriumnetwork/node/core/quality"

	"github.com/stretchr/testify/assert"

	"github.com/mysteriumnetwork/node/identity"
)

type mockMonitoringStatusApi struct {
	response quality.MonitoringStatusResponse
}

func (m *mockMonitoringStatusApi) MonitoringStatus(providerIds []string) quality.MonitoringStatusResponse {
	return m.response
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
		identityLocked bool
		identity       string
		expectedStatus Status
		response       quality.MonitoringStatusResponse
	}{
		{
			identityLocked: false,
			identity:       "0xa",
			expectedStatus: Failed,
			response: map[string]quality.MonitoringStatus{
				"0xa": {
					MonitoringStatus: "failed",
				},
			},
		},
		{
			identityLocked: false,
			identity:       "0xa",
			expectedStatus: Success,
			response: map[string]quality.MonitoringStatus{
				"0xa": {
					MonitoringStatus: "success",
				},
			},
		},
		{
			identityLocked: false,
			identity:       "0xa",
			expectedStatus: Pending,
			response: map[string]quality.MonitoringStatus{
				"0xa": {
					MonitoringStatus: "pending",
				},
			},
		},
		{
			identityLocked: false,
			identity:       "0xa",
			expectedStatus: Unknown,
			response: map[string]quality.MonitoringStatus{
				"0xa": {
					MonitoringStatus: "unknown",
				},
			},
		},
		{
			identityLocked: true,
			identity:       "0xa",
			expectedStatus: Unknown,
			response: map[string]quality.MonitoringStatus{
				"0xa": {
					MonitoringStatus: "success",
				},
			},
		},

		{
			identityLocked: true,
			identity:       "0xa",
			expectedStatus: Unknown,
			response: map[string]quality.MonitoringStatus{
				"not_matching": {
					MonitoringStatus: "success",
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("status %s when %+v", data.expectedStatus, data), func(t *testing.T) {
			fixture := NewStatusTracker(
				newMockCurrentIdentity(data.identity, data.identityLocked),
				&mockMonitoringStatusApi{response: data.response},
			)

			assert.Equal(t, data.expectedStatus, fixture.Status())
		})
	}
}
