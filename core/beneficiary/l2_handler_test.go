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
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/core/storage/boltdb/boltdbtest"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func Test_l2Handler(t *testing.T) {
	path := boltdbtest.CreateTempDir(t)
	defer boltdbtest.RemoveTempDir(t, path)

	st, err := boltdb.NewStorage(path)
	assert.NoError(t, err)

	hermesID := common.HexToAddress("0x0000000000000000000000000000000000000000")
	registryAddr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	chAddr := common.HexToAddress("0x0000000000000000000000000000000000000002")

	id := identity.FromAddress("0x0000000000000000000000000000000000000003")
	benef := common.HexToAddress("0x0000000000000000000000000000000000000004")

	ap := &mockAddressProvider{
		chAddr:   chAddr,
		hermesID: hermesID,
		regAddr:  registryAddr,
	}
	set := &mockSettler{
		expectIdentity: id,
		expectBenef:    chAddr,
		expectHermes:   hermesID,
	}

	l2 := newL2Handler(1, ap, st, set)
	assert.NoError(t, l2.SettleAndSaveBeneficiary(id, benef))

	res, err := l2.GetBeneficiary(id.ToCommonAddress())
	assert.NoError(t, err)
	assert.Equal(t, benef.Hex(), res.Hex())
}

type mockSettler struct {
	expectIdentity identity.Identity
	expectBenef    common.Address
	expectHermes   common.Address
}

func (ma *mockSettler) SettleWithBeneficiary(chainID int64, id identity.Identity, beneficiary, hermesID common.Address) error {
	if id.Address != ma.expectIdentity.Address {
		return errors.New("identity mismatch")
	}
	if beneficiary.Hex() != ma.expectBenef.Hex() {
		return errors.New("beneficiary mismatch")
	}
	if hermesID.Hex() != ma.expectHermes.Hex() {
		return errors.New("hermes mismatch")
	}
	return nil
}

type mockAddressProvider struct {
	chAddr   common.Address
	hermesID common.Address
	regAddr  common.Address
}

func (ma *mockAddressProvider) GetChannelAddress(chainID int64, id identity.Identity) (common.Address, error) {
	return ma.chAddr, nil
}
func (ma *mockAddressProvider) GetActiveHermes(chainID int64) (common.Address, error) {
	return ma.hermesID, nil
}
func (ma *mockAddressProvider) GetRegistryAddress(chainID int64) (common.Address, error) {
	return ma.regAddr, nil
}
