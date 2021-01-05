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
	"time"

	"github.com/mysteriumnetwork/node/core/auth"
)

// AuthRequest request used to authenticate to API.
// swagger:model AuthRequest
type AuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// NewAuthResponse maps to API authentication response.
func NewAuthResponse(jwtToken auth.JWT) AuthResponse {
	return AuthResponse{
		Token:     jwtToken.Token,
		ExpiresAt: jwtToken.ExpirationTime.Format(time.RFC3339),
	}
}

// AuthResponse response after successful authentication to API.
// swagger:model AuthResponse
type AuthResponse struct {
	// example: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6Im15c3QiLCJleHAiOjE2MDEwNDA1NzB9.hnn9FosblyWtx1feSupx3MhZZdkbCuMTaiM6xl54VwQ
	Token string `json:"token"`

	// example: 2019-06-06T11:04:43.910035Z
	ExpiresAt string `json:"expires_at"`
}

// ChangePasswordRequest request used to change API password.
// swagger:model ChangePasswordRequest
type ChangePasswordRequest struct {
	Username    string `json:"username"`
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}
