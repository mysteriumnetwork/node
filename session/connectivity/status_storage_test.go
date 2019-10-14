/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package connectivity

import (
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/stretchr/testify/assert"
)

func TestStatusStorage_AddStatusEntry(t *testing.T) {
	storage := NewStatusStorage()
	e1 := StatusEntry{
		PeerID:       identity.Identity{},
		SessionID:    "1",
		StatusCode:   StatusConnectionOk,
		Message:      "Ok",
		CreatedAtUTC: time.Now().UTC(),
	}
	e2 := StatusEntry{
		PeerID:       identity.Identity{},
		SessionID:    "",
		StatusCode:   StatusConnectionFailed,
		Message:      "Failed",
		CreatedAtUTC: time.Now().UTC().Add(-1 * time.Second),
	}

	storage.AddStatusEntry(e1)
	storage.AddStatusEntry(e2)

	entries := storage.GetAllStatusEntries()
	assert.Len(t, entries, 2)
	assert.Equal(t, e1, entries[0])
	assert.Equal(t, e2, entries[1])
}

func TestStatusStorage_AddStatusEntry_RemovesOldEntries(t *testing.T) {
	storage := NewStatusStorage()
	e1 := StatusEntry{
		SessionID:    "s1",
		CreatedAtUTC: time.Now().UTC().Add(-maxEntriesKeepDuration * 2),
	}
	storage.AddStatusEntry(e1)

	e2 := StatusEntry{
		SessionID:    "s2",
		CreatedAtUTC: time.Now().UTC(),
	}
	storage.AddStatusEntry(e2)

	entries := storage.GetAllStatusEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, e2, entries[0])
}

func TestStatusStorage_GetAllStatusEntries_Returns_Immutable_Data(t *testing.T) {
	storage := NewStatusStorage()
	e1 := StatusEntry{
		PeerID:       identity.Identity{},
		SessionID:    "1",
		StatusCode:   StatusConnectionOk,
		Message:      "Ok",
		CreatedAtUTC: time.Time{},
	}
	storage.AddStatusEntry(e1)

	entries := storage.GetAllStatusEntries()

	entries[0].SessionID = "2"
	assert.NotEqual(t, entries[0].SessionID, storage.(*statusStorage).entries[0].SessionID)
}

func TestStatusStorage_GetAllStatusEntries_Returns_Sorted_Data(t *testing.T) {
	storage := NewStatusStorage()
	e1 := StatusEntry{
		SessionID:    "1",
		CreatedAtUTC: time.Now(),
	}
	e2 := StatusEntry{
		SessionID:    "2",
		CreatedAtUTC: time.Now().Add(-10 * time.Minute),
	}
	e3 := StatusEntry{
		SessionID:    "3",
		CreatedAtUTC: time.Now().Add(15 * time.Minute),
	}
	storage.AddStatusEntry(e1)
	storage.AddStatusEntry(e2)
	storage.AddStatusEntry(e3)

	entries := storage.GetAllStatusEntries()

	assert.Equal(t, e3, entries[0])
	assert.Equal(t, e1, entries[1])
	assert.Equal(t, e2, entries[2])
}
