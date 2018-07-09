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

package metadata

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFormatString(t *testing.T) {
	assert.Equal(
		t,
		"Branch: master. Build id: 1234. Commit: b34a335019aaaae51256d8a33dfdef70ba08b075",
		FormatString("b34a335019aaaae51256d8a33dfdef70ba08b075", "master", "1234"),
	)
}
