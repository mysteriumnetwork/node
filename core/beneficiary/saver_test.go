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
	"io/ioutil"
	"os"
	"testing"

	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/core/storage/boltdb"

	"github.com/mysteriumnetwork/node/identity"

	"github.com/stretchr/testify/assert"
)

func TestBeneficiaryChangeStatus(t *testing.T) {
	// given:
	dir, err := ioutil.TempDir("/tmp", "mysttest")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)
	db, err := boltdb.NewStorage(dir)

	id := identity.FromAddress("0x94bb756322a137a5f0b013dd972d227fe7caa698")
	fixture := beneficiaryChangeKeeper{chainID: 1, st: db}

	// when:
	fixture.updateBeneficiaryChangeStatus(id, Pending, nil)
	r, ok := fixture.BeneficiaryChangeStatus(id)

	// then:
	assert.Equal(t, Pending, r.State)
	assert.True(t, ok)
	assert.Empty(t, r.Error)

	// when:
	fixture.updateBeneficiaryChangeStatus(id, Completed, errors.New("ran out of gas"))
	r, ok = fixture.BeneficiaryChangeStatus(id)

	// then:
	assert.Equal(t, Completed, r.State)
	assert.True(t, ok)
	assert.Equal(t, r.Error, "ran out of gas")

	// when:
	fixture.updateBeneficiaryChangeStatus(id, Completed, nil)
	r, ok = fixture.BeneficiaryChangeStatus(id)

	// then:
	assert.Equal(t, Completed, r.State)
	assert.True(t, ok)
	assert.Empty(t, r.Error)

	// when
	r, ok = fixture.BeneficiaryChangeStatus(identity.FromAddress("0x94aa756322a137a5f0b013dd972d227fe7caa698"))

	// then:
	assert.Nil(t, r)
	assert.True(t, !ok)
}
