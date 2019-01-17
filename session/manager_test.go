/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package session

import (
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

var (
	currentProposalID = 68
	currentProposal   = market.ServiceProposal{
		ID: currentProposalID,
	}
	balanceKeeper   = &mockBalanceTracker{}
	expectedID      = ID("mocked-id")
	expectedSession = Session{
		ID:            expectedID,
		Config:        expectedSessionConfig,
		ConsumerID:    identity.FromAddress("deadbeef"),
		BalanceKeeper: balanceKeeper,
	}
	lastSession Session
)

const expectedSessionConfig = "config_string"

func generateSessionID() (ID, error) {
	return expectedID, nil
}

type fakePromiseProcessor struct {
	started  bool
	proposal market.ServiceProposal
}

func (processor *fakePromiseProcessor) Start(proposal market.ServiceProposal) error {
	processor.started = true
	processor.proposal = proposal
	return nil
}

func (processor *fakePromiseProcessor) Stop() error {
	processor.started = false
	return nil
}

type mockBalanceTracker struct{}

func (mbt *mockBalanceTracker) Track() error {
	return nil
}

func (mbt *mockBalanceTracker) Stop() {

}

func mockBalanceTrackerFactory() BalanceKeeper {
	return balanceKeeper
}

func TestManager_Create_StoresSession(t *testing.T) {
	expectedResult := expectedSession

	sessionStore := NewStorageMemory()
	manager := NewManager(currentProposal, generateSessionID, sessionStore, &fakePromiseProcessor{}, mockBalanceTrackerFactory)

	sessionInstance, err := manager.Create(identity.FromAddress("deadbeef"), currentProposalID, expectedSessionConfig)
	expectedResult.Done = sessionInstance.Done
	assert.NoError(t, err)
	assert.Exactly(t, expectedResult, sessionInstance)
}

func TestManager_Create_RejectsUnknownProposal(t *testing.T) {
	sessionStore := NewStorageMemory()
	manager := NewManager(currentProposal, generateSessionID, sessionStore, &fakePromiseProcessor{}, mockBalanceTrackerFactory)

	sessionInstance, err := manager.Create(identity.FromAddress("deadbeef"), 69, expectedSessionConfig)
	assert.Exactly(t, err, ErrorInvalidProposal)
	assert.Exactly(t, Session{}, sessionInstance)
}

func TestManager_Create_StartsPromiseProcessor(t *testing.T) {
	promiseProcessor := &fakePromiseProcessor{}
	sessionStore := NewStorageMemory()
	manager := NewManager(currentProposal, generateSessionID, sessionStore, promiseProcessor, mockBalanceTrackerFactory)

	_, err := manager.Create(identity.FromAddress("deadbeef"), currentProposalID, expectedSessionConfig)
	assert.NoError(t, err)
	assert.True(t, promiseProcessor.started)
	assert.Exactly(t, currentProposal, promiseProcessor.proposal)
}
