/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/stretchr/testify/assert"
)

func TestLocalBeneficiaryStorage(t *testing.T) {
	// given:
	dir, err := os.MkdirTemp("/tmp", "mysttest")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)
	db, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	localBeneficiaryStorage := NewAddressStorage(db)

	// when
	_, err = localBeneficiaryStorage.Address("random")
	assert.Error(t, err)

	// when
	assert.NoError(t, localBeneficiaryStorage.Save("0xaA11111111111111111111111111111111111111", "0xaaa3333333333333333333333333333333333333"))
	assert.NoError(t, localBeneficiaryStorage.Save("0x2222222222222222222222222222222222222222", "0x6666666666666666666666666666666666666666"))

	addr, err := localBeneficiaryStorage.Address("0xaa11111111111111111111111111111111111111")
	assert.NoError(t, err)
	assert.Equal(t, "0xaaa3333333333333333333333333333333333333", addr)

	addr, err = localBeneficiaryStorage.Address("0x2222222222222222222222222222222222222222")
	assert.NoError(t, err)
	assert.Equal(t, "0x6666666666666666666666666666666666666666", addr)
}
