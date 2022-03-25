/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/ui/versionmanager"
)

// LocalVersionsResponse local version response
// swagger:model LocalVersionsResponse
type LocalVersionsResponse struct {
	Versions []versionmanager.LocalVersion `json:"versions"`
}

// RemoteVersionsResponse local version response
// swagger:model RemoteVersionsResponse
type RemoteVersionsResponse struct {
	Versions []versionmanager.RemoteVersion `json:"versions"`
}

// DownloadNodeUIRequest request for downloading NodeUI version
// swagger:model DownloadNodeUIRequest
type DownloadNodeUIRequest struct {
	Version string `json:"version"`
}

// Valid validate DownloadNodeUIRequest
func (s *DownloadNodeUIRequest) Valid() *apierror.APIError {
	v := apierror.NewValidator()
	if s.Version == "" {
		v.Required("version")
	}
	return v.Err()
}

// SwitchNodeUIRequest request for switching NodeUI version
// swagger:model SwitchNodeUIRequest
type SwitchNodeUIRequest struct {
	Version string `json:"version"`
}

// Valid validate SwitchNodeUIRequest
func (s *SwitchNodeUIRequest) Valid() *apierror.APIError {
	v := apierror.NewValidator()
	if s.Version == "" {
		v.Required("version")
	}
	return v.Err()
}

// UI ui information
// swagger:model UI
type UI struct {
	BundledVersion string `json:"bundled_version"`
	UsedVersion    string `json:"used_version"`
}
