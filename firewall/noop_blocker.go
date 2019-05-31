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

import log "github.com/cihub/seelog"

// NoopVendor is a BlockVendor implementation which only logs allow requests with no effects
// used by default
type NoopVendor struct {
	LogPrefix string
}

func (nb NoopVendor) Reset() {
	log.Info(nb.LogPrefix, "Rules reset was requested")
}

func (nb NoopVendor) BlockOutgoingTraffic() (RemoveRule, error) {
	log.Info(nb.LogPrefix, "Outgoing traffic block requested")
	return nb.logRemoval("Outgoing traffic block removed"), nil
}

// AllowIPAccess logs ip for which access was requested
func (nb NoopVendor) AllowIPAccess(ip string) (RemoveRule, error) {
	log.Info(nb.LogPrefix, "Allow ", ip, " access")
	return nb.logRemoval("Rule for ip: ", ip, " removed"), nil
}

func (nb NoopVendor) logRemoval(vals ...interface{}) RemoveRule {
	return func() {
		vals := append([]interface{}{nb.LogPrefix}, vals...)
		log.Info(vals...)
	}
}

var _ BlockVendor = NoopVendor{}
