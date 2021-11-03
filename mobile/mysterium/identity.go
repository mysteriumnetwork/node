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

package mysterium

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/payments/crypto"
)

// GetIdentityRequest represents identity request.
type GetIdentityRequest struct {
	Address    string
	Passphrase string
}

// GetIdentityResponse represents identity response.
type GetIdentityResponse struct {
	IdentityAddress    string
	ChannelAddress     string
	RegistrationStatus string
}

// GetIdentity finds first identity and unlocks it.
// If there is no identity default one will be created.
func (mb *MobileNode) GetIdentity(req *GetIdentityRequest) (*GetIdentityResponse, error) {
	if req == nil {
		req = &GetIdentityRequest{}
	}

	id, err := mb.identitySelector.UseOrCreate(req.Address, req.Passphrase, mb.chainID)
	if err != nil {
		return nil, fmt.Errorf("could not unlock identity: %w", err)
	}

	channelAddress, err := mb.identityChannelCalculator.GetChannelAddress(mb.chainID, id)
	if err != nil {
		return nil, fmt.Errorf("could not generate channel address: %w", err)
	}

	status, err := mb.identityRegistry.GetRegistrationStatus(mb.chainID, id)
	if err != nil {
		return nil, fmt.Errorf("could not get identity registration status: %w", err)
	}

	return &GetIdentityResponse{
		IdentityAddress:    id.Address,
		ChannelAddress:     channelAddress.Hex(),
		RegistrationStatus: status.String(),
	}, nil
}

// GetIdentityRegistrationFeesResponse represents identity registration fees result.
type GetIdentityRegistrationFeesResponse struct {
	Fee float64
}

// GetIdentityRegistrationFees returns identity registration fees.
func (mb *MobileNode) GetIdentityRegistrationFees() (*GetIdentityRegistrationFeesResponse, error) {
	fees, err := mb.transactor.FetchRegistrationFees(mb.chainID)
	if err != nil {
		return nil, fmt.Errorf("could not get registration fees: %w", err)
	}

	fee := crypto.BigMystToFloat(fees.Fee)

	return &GetIdentityRegistrationFeesResponse{Fee: fee}, nil
}

// RegisterIdentityRequest represents identity registration request.
type RegisterIdentityRequest struct {
	IdentityAddress string
	Token           string
}

// RegisterIdentity starts identity registration in background.
func (mb *MobileNode) RegisterIdentity(req *RegisterIdentityRequest) error {
	fees, err := mb.transactor.FetchRegistrationFees(mb.chainID)
	if err != nil {
		return fmt.Errorf("could not get registration fees: %w", err)
	}

	var token *string
	if req.Token != "" {
		token = &req.Token
	}

	err = mb.transactor.RegisterIdentity(req.IdentityAddress, big.NewInt(0), fees.Fee, "", mb.chainID, token)
	if err != nil {
		return fmt.Errorf("could not register identity: %w", err)
	}

	return nil
}

// IdentityRegistrationChangeCallback represents identity registration status callback.
type IdentityRegistrationChangeCallback interface {
	OnChange(identityAddress string, status string)
}

// RegisterIdentityRegistrationChangeCallback registers callback which is called on identity registration status change.
func (mb *MobileNode) RegisterIdentityRegistrationChangeCallback(cb IdentityRegistrationChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(registry.AppTopicIdentityRegistration, func(e registry.AppEventIdentityRegistration) {
		cb.OnChange(e.ID.Address, e.Status.String())
	})
}

// ExportIdentity exports a given identity address encrypting it with the new passphrase.
func (mb *MobileNode) ExportIdentity(identityAddress, newPassphrase string) ([]byte, error) {
	data, err := mb.identityMover.Export(identityAddress, "", newPassphrase)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ImportIdentity import a given identity address given data (json as string) and the
// current passphrase.
//
// Identity can only be imported if it is registered.
func (mb *MobileNode) ImportIdentity(data []byte, passphrase string) (string, error) {
	identity, err := mb.identityMover.Import(data, passphrase, "")
	if err != nil {
		return "", err
	}

	return identity.Address, nil
}

// IsFreeRegistrationEligible returns true if free registration is possible for a given identity.
func (mb *MobileNode) IsFreeRegistrationEligible(identityAddress string) (bool, error) {
	id := identity.FromAddress(identityAddress)
	ok, err := mb.transactor.GetFreeRegistrationEligibility(id)
	if err != nil {
		return false, err
	}

	return ok, nil
}

// RegistrationTokenReward returns the reward amount for a given token.
func (mb *MobileNode) RegistrationTokenReward(token string) (float64, error) {
	reward, err := mb.transactor.RegistrationTokenReward(token)
	if err != nil {
		return 0, err
	}
	if reward == nil {
		return 0, errors.New("failed to return reward")
	}

	return crypto.BigMystToFloat(reward), nil
}
