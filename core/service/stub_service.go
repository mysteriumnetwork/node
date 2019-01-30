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

package service

import (
	"encoding/json"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session"
)

var _ Service = &serviceFake{}

type serviceFake struct {
	onStartReturnError error
}

func (service *serviceFake) Serve(identity.Identity) error {
	return service.onStartReturnError
}

func (service *serviceFake) Stop() error {
	return nil
}

func (service *serviceFake) GetType() string {
	return "fake"
}

func (service *serviceFake) ProvideConfig(publicKey json.RawMessage) (session.ServiceConfiguration, session.DestroyCallback, error) {
	return struct{}{}, func() {}, nil
}
