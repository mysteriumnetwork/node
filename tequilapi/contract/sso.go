/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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

// MystnodesSSOLinkResponse contains a link to initiate auth via mystnodes
// swagger:model MystnodesSSOLinkResponse
type MystnodesSSOLinkResponse struct {
	Link string `json:"link"`
}

// MystnodesSSOGrantVerificationRequest Mystnodes SSO Grant Verification Request request
type MystnodesSSOGrantVerificationRequest struct {
	AuthorizationGrant    string `json:"authorizationGrant"`
	CodeVerifierBase64url string `json:"codeVerifierBase64url"`
}

// MystnodesSSOGrantLoginRequest Mystnodes SSO Grant Login Request
type MystnodesSSOGrantLoginRequest struct {
	AuthorizationGrant string `json:"authorization_grant"`
}
