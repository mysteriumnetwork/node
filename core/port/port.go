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

package port

import (
	log "github.com/cihub/seelog"
)

// Port (networking)
type Port struct {
	number int
}

// Num returns port's numeric value
func (p *Port) Num() int {
	return p.number
}

// Release releases the port back to the pool.
// After it is called, port is considered disposed and should not be used.
func (p *Port) Release() {
	log.Debug("releasing port back to pool: ", p.Num())
}
