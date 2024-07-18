//go:build !prov_checker

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

package endpoints

import (
	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity/selector"
)

// AddRoutesForConnectionDiag adds proder check route to given router
func AddRoutesForConnectionDiag(
	manager connection.DiagManager,
	stateProvider stateProvider,
	proposalRepository proposalRepository,
	identityRegistry identityRegistry,
	publisher eventbus.Publisher,
	subscriber eventbus.Subscriber,
	addressProvider addressProvider,
	identitySelector selector.Handler,
	options node.Options,
) func(*gin.Engine) error {
	return func(e *gin.Engine) error {
		return nil
	}
}
