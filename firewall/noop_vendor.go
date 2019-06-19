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

package firewall

import "github.com/cihub/seelog"

// NoopVendor is a Vendor implementation which only logs allow requests with no effects
// used by default
type NoopVendor struct {
	LogPrefix string
}

// Reset noop vendor (just log call)
func (nb NoopVendor) Reset() {
	seelog.Info(nb.LogPrefix, "Rules reset was requested")
}

// BlockOutgoingTraffic just logs the call
func (nb NoopVendor) BlockOutgoingTraffic() (RemoveRule, error) {
	seelog.Info(nb.LogPrefix, "Outgoing traffic block requested")
	return nb.logRemoval("Outgoing traffic block removed"), nil
}

// AllowIPAccess logs ip for which access was requested
func (nb NoopVendor) AllowIPAccess(ip string) (RemoveRule, error) {
	seelog.Info(nb.LogPrefix, "Allow ", ip, " access")
	return nb.logRemoval("Rule for ip: ", ip, " removed"), nil
}

func (nb NoopVendor) logRemoval(vals ...interface{}) RemoveRule {
	return func() {
		vals := append([]interface{}{nb.LogPrefix}, vals...)
		seelog.Info(vals...)
	}
}

var _ Vendor = NoopVendor{}
