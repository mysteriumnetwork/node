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
	"reflect"
	"testing"
	"time"

	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/services/openvpn/discovery/dto"
)

func TestProposalToPaymentRate(t *testing.T) {
	tests := []struct {
		name     string
		proposal market.ServiceProposal
		want     dto.PaymentRate
		wantErr  bool
	}{
		{
			name: "accepts WG proposal",
			proposal: market.ServiceProposal{
				PaymentMethod: mockMethod{
					method: "WG",
					price:  money.NewMoney(1, money.CurrencyMyst),
					paymentRate: market.PaymentRate{
						PerTime: time.Minute,
						PerByte: 100,
					},
				},
			},
			wantErr: false,
			want: dto.PaymentRate{
				Price:    money.NewMoney(1, money.CurrencyMyst),
				Duration: time.Minute,
			},
		},
		{
			name: "accepts NOOP proposal",
			proposal: market.ServiceProposal{
				PaymentMethod: mockMethod{
					method: "NOOP",
					price:  money.NewMoney(1, money.CurrencyMyst),
					paymentRate: market.PaymentRate{
						PerTime: time.Minute,
						PerByte: 100,
					},
				},
			},
			wantErr: false,
			want: dto.PaymentRate{
				Price:    money.NewMoney(1, money.CurrencyMyst),
				Duration: time.Minute,
			},
		},
		{
			name: "accepts PER_TIME proposal",
			proposal: market.ServiceProposal{
				PaymentMethod: mockMethod{
					method: "PER_TIME",
					price:  money.NewMoney(1, money.CurrencyMyst),
					paymentRate: market.PaymentRate{
						PerTime: time.Minute,
						PerByte: 100,
					},
				},
			},
			wantErr: false,
			want: dto.PaymentRate{
				Price:    money.NewMoney(1, money.CurrencyMyst),
				Duration: time.Minute,
			},
		},
		{
			name: "rejects unknown proposal",
			proposal: market.ServiceProposal{
				PaymentMethod: mockMethod{
					method: "unknown",
				},
			},
			wantErr: true,
		},
		{
			name: "rejects PER_TIME proposal with zero time",
			proposal: market.ServiceProposal{
				PaymentMethod: mockMethod{
					method: "PER_TIME",
					price:  money.NewMoney(1, money.CurrencyMyst),
					paymentRate: market.PaymentRate{
						PerTime: 0,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProposalToPaymentRate(tt.proposal)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProposalToPaymentRate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ProposalToPaymentRate() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockMethod struct {
	method      string
	price       money.Money
	paymentRate market.PaymentRate
}

func (method mockMethod) GetPrice() money.Money       { return method.price }
func (method mockMethod) GetType() string             { return method.method }
func (method mockMethod) GetRate() market.PaymentRate { return method.paymentRate }
