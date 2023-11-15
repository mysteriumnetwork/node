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

package beneficiary

import (
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/mysteriumnetwork/node/identity"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestBeneficiaryChangeStatus(t *testing.T) {
	// given:
	dir, err := os.MkdirTemp("/tmp", "mysttest")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)
	db, err := boltdb.NewStorage(dir)

	id := identity.FromAddress("0x94bb756322a137a5f0b013dd972d227fe7caa698")
	benef := "0x94bb756322a137a5f0b013dd972d227fe7caa000"
	fixture := beneficiaryChangeKeeper{chainID: 1, st: db}

	// when:
	fixture.updateChangeStatus(id, Pending, benef, nil)
	r, err := fixture.GetChangeStatus(id)

	// then:
	assert.Equal(t, Pending, r.State)
	assert.NoError(t, err)
	assert.Empty(t, r.Error)

	// when:
	fixture.updateChangeStatus(id, Completed, benef, errors.New("ran out of gas"))
	r, err = fixture.GetChangeStatus(id)

	// then:
	assert.Equal(t, Completed, r.State)
	assert.NoError(t, err)
	assert.Equal(t, r.Error, "ran out of gas")

	// when:
	fixture.updateChangeStatus(id, Completed, benef, nil)
	r, err = fixture.GetChangeStatus(id)

	// then:
	assert.Equal(t, Completed, r.State)
	assert.NoError(t, err)
	assert.Empty(t, r.Error)

	// when
	r, err = fixture.GetChangeStatus(identity.FromAddress("0x94aa756322a137a5f0b013dd972d227fe7caa698"))

	// then:
	assert.Nil(t, r)
	assert.Error(t, err)

	// when:
	fixture.updateChangeStatus(id, Pending, benef, nil)
	r, err = fixture.GetChangeStatus(id)
	assert.NoError(t, err)
	assert.Equal(t, Pending, r.State)

	// then:
	r, err = fixture.CleanupAndGetChangeStatus(id, benef)
	assert.NoError(t, err)
	assert.Equal(t, Completed, r.State)

}
