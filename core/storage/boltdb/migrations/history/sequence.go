/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package history

import (
	"time"

	"github.com/mysteriumnetwork/node/core/storage/boltdb/migrations"
)

// Sequence contains the whole migration sequence for boltdb
var Sequence = []migrations.Migration{
	{
		Name: "session-to-session-history",
		Date: time.Date(
			2018, 12, 04, 12, 00, 00, 0, time.UTC),
		Migrate: migrations.MigrateSessionToHistory,
	},
	{
		Name: "settlements-to-rows",
		Date: time.Date(
			2020, 8, 17, 14, 27, 00, 0, time.UTC),
		Migrate: migrations.SettlementValuesToRows,
	},
	{
		Name: "registration-status-to-new",
		Date: time.Date(
			2021, 3, 15, 16, 00, 00, 0, time.UTC),
		Migrate: migrations.MigrateRegistrationState,
	},
	{
		Name: "registration-status-to-new-mainnet",
		Date: time.Date(
			2021, 10, 11, 0, 00, 00, 0, time.UTC),
		Migrate: migrations.MigrateRegistrationState,
	},
}
