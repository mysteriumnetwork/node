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

package pingpong

import (
	"fmt"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
)

// ProposalToPaymentRate parses the proposal and converts it to payment method
func ProposalToPaymentRate(proposal market.ServiceProposal) (dto.PaymentRate, error) {
	switch proposal.PaymentMethod.GetType() {
	case "WG":
		fallthrough
	case "NOOP":
		fallthrough
	case "PER_TIME":
		time := proposal.PaymentMethod.GetRate().PerTime
		if time == 0 {
			return dto.PaymentRate{}, fmt.Errorf("unsupported payment per time %q", time)
		}
		return dto.PaymentRate{
			Price:    proposal.PaymentMethod.GetPrice(),
			Duration: proposal.PaymentMethod.GetRate().PerTime,
		}, nil
	default:
		return dto.PaymentRate{}, fmt.Errorf("unsupported payment method %q", proposal.PaymentMethod.GetType())
	}
}
