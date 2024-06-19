/*
 * Copyright (C) 2024 The "MysteriumNetwork/node" Authors.
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

package connection

import (
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/rs/zerolog/log"
)

// Diag is used to start provider check
func Diag(cm *diagConnectionManager, con *conn, providerID string) {
	c, ok := con.activeConnection.(ConnectionDiag)
	res := false
	if ok {
		log.Debug().Msgf("Check provider> %v", providerID)

		res = c.Diag()
		cm.DisconnectSingle(con)
	}
	ev := quality.DiagEvent{ProviderID: providerID, Result: res}
	con.resChannel <- ev
	close(con.resChannel)
}
