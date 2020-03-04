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

package traversal

// NoopPinger does nothing
type NoopPinger struct{}

// NewNoopPinger returns noop nat pinger
func NewNoopPinger() NATPinger {
	return &NoopPinger{}
}

// Valid returns that noop pinger is not a valid pinger
func (np *NoopPinger) Valid() bool {
	return false
}

// StopNATProxy does nothing
func (np *NoopPinger) StopNATProxy() {}

// SetProtectSocketCallback does nothing
func (np *NoopPinger) SetProtectSocketCallback(socketProtect func(socket int) bool) {}

// Start does nothing
func (np *NoopPinger) Start() {}

// Stop does nothing
func (np *NoopPinger) Stop() {}

// PingProvider does nothing
func (np *NoopPinger) PingProvider(_ string, _, _ []int, proxyPort int) (int, int, error) {
	return 0, 0, nil
}

// PingTarget does nothing
func (np *NoopPinger) PingTarget(Params) {}

// BindServicePort does nothing
func (np *NoopPinger) BindServicePort(key string, port int) {}
