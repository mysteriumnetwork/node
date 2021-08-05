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
	"github.com/mysteriumnetwork/node/identity"
)

func (c *cliApp) stake(args []string) (err error) {
	var usage = strings.Join([]string{
		"Usage: stake <action> [args]",
		"Available actions:",
		"  " + usageIncreaseStake,
		"  " + usageDecreaseStake,
	}, "\n")

	if len(args) == 0 {
		clio.Info(usage)
		return errWrongArgumentCount
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "increase":
		return c.increaseStake(actionArgs)
	case "decrease":
		return c.decreaseStake(actionArgs)
	default:
		fmt.Println(usage)
		return errUnknownSubCommand(args[0])
	}
}

const usageIncreaseStake = "increase <identity>"
const usageDecreaseStake = "decrease <identity> <amount>"

func (c *cliApp) decreaseStake(args []string) (err error) {
	if len(args) != 2 {
		clio.Info("Usage: " + usageDecreaseStake)
		return errWrongArgumentCount
	}

	res, ok := new(big.Int).SetString(args[1], 10)
	if !ok {
		return fmt.Errorf("could not parse amount: %v", args[1])
	}

	err = c.tequilapi.DecreaseStake(identity.FromAddress(args[0]), res)
	if err != nil {
		return fmt.Errorf("could not decrease stake: %w", err)
	}
	return nil
}

func (c *cliApp) increaseStake(args []string) (err error) {
	if len(args) != 1 {
		clio.Info("Usage: " + usageIncreaseStake)
		return errWrongArgumentCount
	}

	hermesID, err := c.config.GetHermesID()
	if err != nil {
		return fmt.Errorf("could not get Hermes ID: %w", err)
	}
	clio.Info("Waiting for settlement to complete")
	errChan := make(chan error)

	go func() {
		errChan <- c.tequilapi.SettleIntoStake(identity.FromAddress(args[0]), identity.FromAddress(hermesID), true)
	}()

	timeout := time.After(time.Minute * 2)
	for {
		select {
		case <-timeout:
			fmt.Println()
			return errTimeout
		case <-time.After(time.Millisecond * 500):
			fmt.Print(".")
		case err := <-errChan:
			fmt.Println()
			if err != nil {
				return fmt.Errorf("settlement failed: %w", err)
			}
			clio.Info("settlement succeeded")
			return nil
		}
	}
}
