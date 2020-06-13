/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package netutil

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/mysteriumnetwork/node/core/storage/boltdb"
	"github.com/stretchr/testify/assert"
)

func TestClearStaleRoutes(t *testing.T) {
	t.Run("No panic on clear without init", func(t *testing.T) {
		ClearStaleRoutes()
	})

	dir, err := ioutil.TempDir("/tmp", "mysttest")
	assert.NoError(t, err)

	defer os.RemoveAll(dir)

	db, err := boltdb.NewStorage(dir)
	assert.NoError(t, err)
	SetRouteManagerStorage(db)

	defaultRouteManager.deleteRoute = noopDeleteRoute

	t.Run("record deleted", func(t *testing.T) {
		err := db.Store(routeRecordBucket, &route{Record: "8.9.7.6|1.2.3.4"})
		assert.NoError(t, err)
		err = db.Store(routeRecordBucket, &route{Record: "4.5.7.6|8.9.3.4"})
		assert.NoError(t, err)
		err = db.Store(routeRecordBucket, &route{Record: "1.2.7.6|3.6.5.4"})
		assert.NoError(t, err)

		var records []route
		err = defaultRouteManager.db.GetAllFrom(routeRecordBucket, &records)
		assert.NoError(t, err)
		assert.Len(t, records, 3)

		ClearStaleRoutes()

		err = defaultRouteManager.db.GetAllFrom(routeRecordBucket, &records)
		assert.NoError(t, err)
		assert.Len(t, records, 0)
	})
}

func noopDeleteRoute(ip, wg string) error {
	return nil
}
