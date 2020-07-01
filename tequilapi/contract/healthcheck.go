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

// HealthCheckDTO holds API healthcheck.
// swagger:model HealthCheckDTO
type HealthCheckDTO struct {
	// example: 25h53m33.540493171s
	Uptime string `json:"uptime"`

	// example: 10449
	Process int `json:"process"`

	// example: 0.0.6
	Version   string       `json:"version"`
	BuildInfo BuildInfoDTO `json:"build_info"`
}

// BuildInfoDTO holds info about build.
// swagger:model BuildInfoDTO
type BuildInfoDTO struct {
	// example: <unknown>
	Commit string `json:"commit"`

	// example: <unknown>
	Branch string `json:"branch"`

	// example: dev-build
	BuildNumber string `json:"build_number"`
}
