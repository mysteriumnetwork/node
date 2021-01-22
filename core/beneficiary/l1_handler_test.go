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

package beneficiary

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func Test_l1Handler(t *testing.T) {
	hermesID := common.HexToAddress("0x0000000000000000000000000000000000000000")
	registryAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")

	id := identity.FromAddress("0x0000000000000000000000000000000000000003")
	benef := common.HexToAddress("0x0000000000000000000000000000000000000004")

	ap := &mockAddressProvider{
		hermesID: hermesID,
		regAddr:  registryAddr,
	}

	set := &mockSettler{
		expectBenef:    benef,
		expectIdentity: id,
		expectHermes:   hermesID,
	}

	bc := &mockMultichainBC{
		returnBenef:    benef,
		expectRegistry: registryAddr,
	}

	l1 := newL1Handler(1, ap, bc, set)
	assert.NoError(t, l1.SettleAndSaveBeneficiary(id, benef))

	res, err := l1.GetBeneficiary(id.ToCommonAddress())
	assert.NoError(t, err)
	assert.Equal(t, benef.Hex(), res.Hex())
}

type mockMultichainBC struct {
	returnBenef    common.Address
	expectRegistry common.Address
}

func (m *mockMultichainBC) GetBeneficiary(chainID int64, registryAddress, identity common.Address) (common.Address, error) {
	if m.expectRegistry != registryAddress {
		return common.Address{}, errors.New("registry mismatch")
	}
	return m.returnBenef, nil
}
