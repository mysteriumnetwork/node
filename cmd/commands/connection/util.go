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
	"math/big"
	"time"

	"github.com/mysteriumnetwork/node/datasize"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

func proposalFormatted(p *contract.ProposalDTO) string {
	ppm, ppgb := getPrices(p.PaymentMethod)
	return fmt.Sprintf("| Identity: %s\t| Type: %s\t| Country: %s\t| Price: %s\t%s\t|",
		p.ProviderID,
		p.ServiceDefinition.LocationOriginate.NodeType,
		p.ServiceDefinition.LocationOriginate.Country,
		ppm,
		ppgb,
	)
}

func getPrices(pm contract.PaymentMethodDTO) (string, string) {
	ppm := aproxPricePerHour(pm)
	ppgb := aproxPricePerGB(pm)
	if ppm.Amount.Cmp(big.NewInt(0)) == 0 &&
		ppgb.Amount.Cmp(big.NewInt(0)) == 0 {
		return "Free", ""
	}

	return fmt.Sprintf("%s/hour", ppm.String()),
		fmt.Sprintf("%s/GB", ppgb.String())
}

func aproxPricePerHour(pm contract.PaymentMethodDTO) money.Money {
	s := time.Duration(pm.Rate.PerSeconds) * time.Second
	min := new(big.Float).SetFloat64(s.Minutes())
	if min.Cmp(big.NewFloat(0)) == 0 {
		return money.New(new(big.Int).SetInt64(0), pm.Price.Currency)
	}

	price := new(big.Float).SetInt(pm.Price.Amount)
	perMinute := new(big.Float).Quo(price, min)
	perMinuteRounded, _ := perMinute.Int(nil)

	perHour := new(big.Int).Mul(perMinuteRounded, new(big.Int).SetInt64(60))
	return money.New(perHour, pm.Price.Currency)
}

func aproxPricePerGB(pm contract.PaymentMethodDTO) money.Money {
	gb := new(big.Float).Quo(
		new(big.Float).SetUint64(pm.Rate.PerBytes),
		new(big.Float).SetUint64(datasize.GiB.Bytes()),
	)
	if gb.Cmp(big.NewFloat(0)) == 0 {
		return money.New(new(big.Int).SetInt64(0), pm.Price.Currency)
	}

	price := new(big.Float).SetInt(pm.Price.Amount)
	perGB := new(big.Float).Quo(price, gb)
	perGBRounded, _ := perGB.Int(nil)
	return money.New(perGBRounded, pm.Price.Currency)
}
