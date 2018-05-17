/*
 * Copyright (C) 2018 The Mysterium Network Authors
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

package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_LoadIdentityExisting(t *testing.T) {
	identityHandler := &handlerFake{}
	id, err := LoadIdentity(identityHandler, "existing", "")
	assert.Equal(t, "existing", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityLast(t *testing.T) {
	identityHandler := &handlerFake{LastAddress: "last"}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "last", id.Address)
	assert.Nil(t, err)
}

func Test_LoadIdentityNew(t *testing.T) {
	identityHandler := &handlerFake{}
	id, err := LoadIdentity(identityHandler, "", "")
	assert.Equal(t, "new", id.Address)
	assert.Nil(t, err)
}
