/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package event

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
)

const (
	// AppTopicSession represents the session change topic.
	AppTopicSession = "Session change"
	// AppTopicDataTransferred represents the data transfer topic.
	AppTopicDataTransferred = "Session data transferred"
	// AppTopicTokensEarned is a topic for publish events about tokens earned as a provider.
	AppTopicTokensEarned = "SessionTokensEarned"
)

// AppEventDataTransferred represents the data transfer event
type AppEventDataTransferred struct {
	ID       string
	Up, Down uint64
}

// AppEventTokensEarned is an update on tokens earned during current session
type AppEventTokensEarned struct {
	ProviderID identity.Identity
	SessionID  string
	Total      *big.Int
}

// Status represents the different actions that might happen on a session
type Status string

const (
	// CreatedStatus indicates a session has been created
	CreatedStatus Status = "CreatedStatus"
	// RemovedStatus indicates a session has been removed
	RemovedStatus Status = "RemovedStatus"
	// AcknowledgedStatus indicates a session has been reported as a success from consumer side
	AcknowledgedStatus Status = "AcknowledgedStatus"
)

// AppEventSession represents the session change payload
type AppEventSession struct {
	Status  Status
	Service ServiceContext
	Session SessionContext
}

// ServiceContext holds service context metadata
type ServiceContext struct {
	ID string
}

// SessionContext holds session context metadata
type SessionContext struct {
	ID               string
	StartedAt        time.Time
	ConsumerID       identity.Identity
	ConsumerLocation market.Location
	HermesID         common.Address
	Proposal         market.ServiceProposal
}
