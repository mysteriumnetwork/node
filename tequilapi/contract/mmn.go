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

package contract

import (
	"github.com/mysteriumnetwork/node/tequilapi/validation"
)

// MMNApiKeyRequest request used to manage MMN's API key.
// swagger:model MMNApiKeyRequest
type MMNApiKeyRequest struct {
	ApiKey string `json:"api_key"`
}

// Validate validates fields in request
func (r MMNApiKeyRequest) Validate() *validation.FieldErrorMap {
	errs := validation.NewErrorMap()
	if len(r.ApiKey) == 0 {
		errs.ForField("api_key").Required()
	}
	if len(r.ApiKey) < 40 {
		errs.ForField("api_key").Invalid("Invalid API key")
	}
	return errs
}
