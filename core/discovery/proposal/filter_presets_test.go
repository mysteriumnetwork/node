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

package proposal

import (
	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/stretchr/testify/assert"

	"os"
	"testing"
)

func Test_ProposalFilterPreset(t *testing.T) {
	dir, err := os.MkdirTemp("", "consumerTotalsStorageTest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	bolt, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	presetStorage := NewFilterPresetStorage(bolt)

	t.Run("list", func(t *testing.T) {
		ls, err := presetStorage.List()
		assert.NoError(t, err)

		for _, n := range []string{"Media Streaming", "Browsing", "Download"} {
			_, ok := ls.byName(n)
			assert.Truef(t, ok, "Missing '%s' in default presets", n)
		}
	})

	t.Run("save", func(t *testing.T) {
		// given
		err := presetStorage.Save(FilterPreset{Name: "Boink"})
		assert.NoError(t, err)
		err = presetStorage.Save(FilterPreset{Name: "Boink 2"})
		assert.NoError(t, err)

		// when
		ls, err := presetStorage.List()
		assert.NoError(t, err)

		// then
		p1, ok := ls.byName("Boink")
		assert.True(t, ok)
		assert.Equal(t, startingID, p1.ID)

		p2, ok := ls.byName("Boink 2")
		assert.True(t, ok)
		assert.Equal(t, p1.ID+1, p2.ID)

		// and
		p2.Name = "Boink 3"
		err = presetStorage.Save(p2)
		assert.NoError(t, err)

		ls, err = presetStorage.List()
		assert.NoError(t, err)
		_, ok = ls.byName("Boink 2")
		assert.False(t, ok)

		p3, ok := ls.byName("Boink 3")
		assert.True(t, ok)
		assert.Equal(t, p2.ID, p3.ID)
	})

	t.Run("delete", func(t *testing.T) {
		// given
		err := presetStorage.Save(FilterPreset{Name: "Delete Me"})
		assert.NoError(t, err)

		ls, err := presetStorage.List()
		assert.NoError(t, err)

		p, ok := ls.byName("Delete Me")
		assert.True(t, p.ID >= startingID)
		assert.True(t, ok)

		// when
		err = presetStorage.Delete(p.ID)
		assert.NoError(t, err)

		// then
		ls, err = presetStorage.List()
		assert.NoError(t, err)
		_, ok = ls.byName("Delete Me")
		assert.False(t, ok)
	})
}

func (w *FilterPresets) byName(name string) (FilterPreset, bool) {
	for _, p := range w.Entries {
		if p.Name == name {
			return p, true
		}
	}
	return FilterPreset{}, false
}
