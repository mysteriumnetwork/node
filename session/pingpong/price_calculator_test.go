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
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
)

func Test_isServiceFree(t *testing.T) {
	tests := []struct {
		name   string
		method market.PaymentMethod
		want   bool
	}{
		{
			name: "not free if only time payment is set",
			method: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerTime: time.Minute},
			},
			want: false,
		},
		{
			name: "not free if only byte payment is set",
			method: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerByte: 1},
			},
			want: false,
		},
		{
			name: "not free if time + byte payment is set",
			method: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerByte: 1, PerTime: time.Minute},
			},
			want: false,
		},
		{
			name:   "free if nil",
			method: nil,
			want:   true,
		},
		{
			name: "free if both zero",
			method: &mockPaymentMethod{
				price: money.NewMoney(10, money.CurrencyMyst),
				rate:  market.PaymentRate{PerByte: 0, PerTime: 0},
			},
			want: true,
		},
		{
			name: "free if price zero",
			method: &mockPaymentMethod{
				price: money.NewMoney(0, money.CurrencyMyst),
				rate:  market.PaymentRate{PerByte: 1, PerTime: 2},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isServiceFree(tt.method); got != tt.want {
				t.Errorf("isServiceFree() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_calculatePaymentAmount(t *testing.T) {
	type args struct {
		timePassed      time.Duration
		bytesTransfered dataTransfered
		method          market.PaymentMethod
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "returns zero on free service",
			args: args{
				timePassed: time.Hour,
				bytesTransfered: dataTransfered{
					up: 100, down: 100,
				},
				method: &mockPaymentMethod{
					price: money.NewMoney(0, money.CurrencyMyst),
					rate:  market.PaymentRate{PerByte: 1, PerTime: 2},
				},
			},
			want: 0,
		},
		{
			name: "calculates time only",
			args: args{
				timePassed: time.Hour,
				bytesTransfered: dataTransfered{
					up: 100, down: 100,
				},
				method: &mockPaymentMethod{
					price: money.NewMoney(50000, money.CurrencyMyst),
					rate:  market.PaymentRate{PerByte: 0, PerTime: time.Minute},
				},
			},
			want: 60 * 50000,
		},
		{
			name: "calculates time only with seconds",
			args: args{
				timePassed: time.Hour,
				bytesTransfered: dataTransfered{
					up: 100, down: 100,
				},
				method: &mockPaymentMethod{
					price: money.NewMoney(50000, money.CurrencyMyst),
					rate:  market.PaymentRate{PerByte: 0, PerTime: time.Second},
				},
			},
			want: 60 * 60 * 50000,
		},
		{
			name: "calculates bytes only",
			args: args{
				timePassed: time.Hour,
				bytesTransfered: dataTransfered{
					up: 1000000000 / 2, down: 1000000000 / 2,
				},
				method: &mockPaymentMethod{
					price: money.NewMoney(7000000, money.CurrencyMyst),
					rate:  market.PaymentRate{PerByte: 1000000000, PerTime: 0},
				},
			},
			want: 7000000,
		},
		{
			name: "calculates both",
			args: args{
				timePassed: time.Hour,
				bytesTransfered: dataTransfered{
					up: 1000000000 / 2, down: 1000000000 / 2,
				},
				method: &mockPaymentMethod{
					price: money.NewMoney(50000, money.CurrencyMyst),
					rate:  market.PaymentRate{PerByte: 7142857, PerTime: time.Minute},
				},
			},
			// 7000000 is the price per gigabyte
			// 50000 is the time per minute, 60 is the amount of minutes
			want: 7000000 + 60*50000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculatePaymentAmount(tt.args.timePassed, tt.args.bytesTransfered, tt.args.method); got != tt.want {
				t.Errorf("calculatePaymentAmount() = %v, want %v", got, tt.want)
			}
		})
	}
}
