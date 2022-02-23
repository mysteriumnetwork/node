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

package connection

import (
	"fmt"

	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func proposalFormatted(p *contract.ProposalDTO) string {
	return fmt.Sprintf("| Identity: %s\t| Type: %s\t| Country: %s\t | Price: %s/hour\t%s/GiB\t|",
		p.ProviderID,
		p.Location.IPType,
		p.Location.Country,
		p.Price.PerHourTokens.Human,
		p.Price.PerGiBTokens.Human,
	)
}
