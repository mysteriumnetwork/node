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
	"math/big"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/market"
)

func Test_isServiceFree(t *testing.T) {
	tests := []struct {
		name  string
		price *market.Price
		want  bool
	}{
		{
			name:  "not free if only time payment is set",
			price: market.NewPrice(10, 0),
			want:  false,
		},
		{
			name:  "not free if only byte payment is set",
			price: market.NewPrice(0, 10),
			want:  false,
		},
		{
			name:  "not free if time + byte payment is set",
			price: market.NewPrice(10, 10),
			want:  false,
		},
		{
			name:  "free if empty",
			price: market.NewPrice(0, 0),
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.price.IsFree(); got != tt.want {
				t.Errorf("isServiceFree() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_CalculatePaymentAmount(t *testing.T) {
	type args struct {
		timePassed       time.Duration
		bytesTransferred DataTransferred
		price            *market.Price
	}
	tests := []struct {
		name string
		args args
		want *big.Int
	}{
		{
			name: "returns zero on free service",
			args: args{
				timePassed: time.Hour,
				bytesTransferred: DataTransferred{
					Up: 100, Down: 100,
				},
				price: market.NewPrice(0, 0),
			},
			want: big.NewInt(0),
		},
		{
			name: "calculates time only",
			args: args{
				timePassed: time.Hour,
				bytesTransferred: DataTransferred{
					Up: 100, Down: 100,
				},
				price: market.NewPrice(3000000, 0),
			},
			want: big.NewInt(60 * 50000),
		},
		{
			name: "calculates bytes only",
			args: args{
				timePassed: time.Hour,
				bytesTransferred: DataTransferred{
					Up: datasize.GiB.Bytes() / 2, Down: datasize.GiB.Bytes() / 2,
				},
				price: market.NewPrice(0, 7000000),
			},
			want: big.NewInt(7000000),
		},
		{
			name: "calculates both",
			args: args{
				timePassed: time.Hour,
				bytesTransferred: DataTransferred{
					Up: datasize.GiB.Bytes() / 2, Down: datasize.GiB.Bytes() / 2,
				},
				price: market.NewPrice(3000000, 7000000),
			},
			// 7000000 is the price per gibibyte, 3000000 is the price per hour
			want: big.NewInt(7000000 + 3000000),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculatePaymentAmount(tt.args.timePassed, tt.args.bytesTransferred, *tt.args.price); got.Cmp(tt.want) != 0 {
				t.Errorf("CalculatePaymentAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}
