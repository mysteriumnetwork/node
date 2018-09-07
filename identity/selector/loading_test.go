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

package selector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_LoadIdentityExisting(t *testing.T) {
	loadIdentity := NewLoader(&handlerFake{}, "existing", "")

	id, err := loadIdentity()
	assert.Equal(t, "existing", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityLast(t *testing.T) {
	loadIdentity := NewLoader(&handlerFake{LastAddress: "last"}, "", "")

	id, err := loadIdentity()
	assert.Equal(t, "last", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityNew(t *testing.T) {
	loadIdentity := NewLoader(&handlerFake{}, "", "")

	id, err := loadIdentity()
	assert.Equal(t, "new", id.Address)
	assert.Nil(t, err)
}
