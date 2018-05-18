/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
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

package dialog

import (
	"github.com/mysterium/node/communication"
)

// Consume is trying to establish new dialog with Provider
const endpointDialogCreate = communication.RequestEndpoint("dialog-create")

var (
	responseOK              = dialogCreateResponse{200, "OK"}
	responseInvalidIdentity = dialogCreateResponse{400, "Invalid identity"}
	responseInternalError   = dialogCreateResponse{500, "Failed to create dialog"}
)

type dialogCreateRequest struct {
	PeerID string `json:"peer_id"`
}

type dialogCreateResponse struct {
	Reason        uint   `json:"reason"`
	ReasonMessage string `json:"reasonMessage"`
}
