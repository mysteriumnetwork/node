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
	"math/big"
	"testing"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/market"
	"github.com/stretchr/testify/assert"
)

func TestValidator_Validate(t *testing.T) {
	type fields struct {
		consumerBalanceGetter consumerBalanceGetter
		unlockChecker         unlockChecker
	}
	type args struct {
		consumerID identity.Identity
		price      market.Price
		chainID    int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name:    "returns insufficient balance",
			wantErr: ErrInsufficientBalance,
			fields: fields{
				unlockChecker: &mockUnlockChecker{
					toReturn: true,
				},
				consumerBalanceGetter: &mockConsumerBalanceGetter{
					toReturn:    big.NewInt(1),
					forceReturn: big.NewInt(99),
				},
			},
			args: args{
				chainID:    1,
				consumerID: identity.FromAddress("whatever"),
				price: market.Price{
					PricePerHour: big.NewInt(6000),
					PricePerGiB:  big.NewInt(6000),
				},
			},
		},
		{
			name:    "resync balance on insufficient balance",
			wantErr: nil,
			fields: fields{
				unlockChecker: &mockUnlockChecker{
					toReturn: true,
				},
				consumerBalanceGetter: &mockConsumerBalanceGetter{
					toReturn:    big.NewInt(99),
					forceReturn: big.NewInt(100),
				},
			},
			args: args{
				chainID:    1,
				consumerID: identity.FromAddress("whatever"),
				price: market.Price{
					PricePerHour: big.NewInt(600),
					PricePerGiB:  big.NewInt(600),
				},
			},
		},
		{
			name:    "returns unlock required",
			wantErr: ErrUnlockRequired,
			fields: fields{
				unlockChecker: &mockUnlockChecker{
					toReturn: false,
				},
			},
			args: args{
				chainID:    1,
				consumerID: identity.FromAddress("whatever"),
			},
		},
		{
			name:    "returns no error if conditions are satisfied",
			wantErr: nil,
			fields: fields{
				unlockChecker: &mockUnlockChecker{
					toReturn: true,
				},
				consumerBalanceGetter: &mockConsumerBalanceGetter{
					toReturn: big.NewInt(101),
				},
			},
			args: args{
				chainID:    1,
				consumerID: identity.FromAddress("whatever"),
				price: market.Price{
					PricePerHour: big.NewInt(100),
					PricePerGiB:  big.NewInt(100),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Validator{
				consumerBalanceGetter: tt.fields.consumerBalanceGetter,
				unlockChecker:         tt.fields.unlockChecker,
			}
			err := v.Validate(tt.args.chainID, tt.args.consumerID, tt.args.price)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error(), tt.name)
			} else {
				assert.NoError(t, err, tt.name)
			}
		})
	}
}

type mockUnlockChecker struct {
	toReturn bool
}

func (muc *mockUnlockChecker) IsUnlocked(id string) bool {
	return muc.toReturn
}

type mockConsumerBalanceGetter struct {
	needSync    bool
	toReturn    *big.Int
	forceReturn *big.Int
}

func (mcbg *mockConsumerBalanceGetter) NeedsForceSync(chainID int64, id identity.Identity) bool {
	return mcbg.needSync
}

func (mcbg *mockConsumerBalanceGetter) GetBalance(chainID int64, id identity.Identity) *big.Int {
	return mcbg.toReturn
}

func (mcbg *mockConsumerBalanceGetter) ForceBalanceUpdate(chainID int64, id identity.Identity) *big.Int {
	return mcbg.forceReturn
}
