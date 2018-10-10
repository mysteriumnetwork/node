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

package identity

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// BalanceRegistry provides amount of money that identity have on the balance in the blockchain
type BalanceRegistry func(Identity) (uint64, error)

// NewBalanceRegistry creates new balance registry
func NewBalanceRegistry(etherClient *ethclient.Client) BalanceRegistry {
	return func(identity Identity) (uint64, error) {
		balance, err := etherClient.BalanceAt(context.Background(), common.HexToAddress(identity.Address), nil)
		return balance.Uint64(), err
	}
}
