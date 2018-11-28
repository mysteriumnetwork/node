// +build !darwin,!windows,!linux

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

package cmd

import (
	"errors"

	"github.com/mysteriumnetwork/node/core/node"
)

// ErrServiceStartingUnsupported represents the error when this entrypoint is used in an unsupported OS
var ErrServiceStartingUnsupported = errors.New("running of services is not supported on your OS")

// BootstrapServices loads all the components required for running services
func (di *Dependencies) BootstrapServices(nodeOptions node.Options) error {
	return ErrServiceStartingUnsupported
}
