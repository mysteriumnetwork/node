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

package cli

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/identity"
)

func (c *cliApp) stake(argsString string) {
	var usage = strings.Join([]string{
		"Usage: stake <action> [args]",
		"Available actions:",
		"  " + usageIncreaseStake,
		"  " + usageDecreaseStake,
	}, "\n")

	if len(argsString) == 0 {
		clio.Info(usage)
		return
	}

	args := strings.Fields(argsString)
	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "increase":
		c.increaseStake(actionArgs)
	case "decrease":
		c.decreaseStake(actionArgs)
	default:
		clio.Warnf("Unknown sub-command '%s'\n", argsString)
		fmt.Println(usage)
	}
}

const usageIncreaseStake = "increase <identity>"
const usageDecreaseStake = "decrease <identity> <amount>"

func (c *cliApp) decreaseStake(args []string) {
	if len(args) != 2 {
		clio.Info("Usage: " + usageDecreaseStake)
		return
	}

	res, ok := new(big.Int).SetString(args[1], 10)
	if !ok {
		clio.Warn("could not parse amount")
		return
	}

	fees, err := c.tequilapi.GetTransactorFees()
	if err != nil {
		clio.Warn("could not get transactor fee: ", err)
		return
	}

	err = c.tequilapi.DecreaseStake(identity.FromAddress(args[0]), res, fees.DecreaseStake)
	if err != nil {
		clio.Warn("could not decrease stake: ", err)
		return
	}
}

func (c *cliApp) increaseStake(args []string) {
	if len(args) != 1 {
		clio.Info("Usage: " + usageIncreaseStake)
		return
	}

	accountantID := c.config.GetStringByFlag(config.FlagHermesID)
	clio.Info("Waiting for settlement to complete")
	errChan := make(chan error)

	go func() {
		errChan <- c.tequilapi.SettleIntoStake(identity.FromAddress(args[0]), identity.FromAddress(accountantID), true)
	}()

	timeout := time.After(time.Minute * 2)
	for {
		select {
		case <-timeout:
			fmt.Println()
			clio.Warn("Settlement timed out")
			return
		case <-time.After(time.Millisecond * 500):
			fmt.Print(".")
		case err := <-errChan:
			fmt.Println()
			if err != nil {
				clio.Warn("settlement failed: ", err.Error())
				return
			}
			clio.Info("settlement succeeded")
			return
		}
	}
}
