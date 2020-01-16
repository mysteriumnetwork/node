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
	"strconv"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/pkg/errors"
)

func (c *cliApp) identities(argsString string) {
	var usage = strings.Join([]string{
		"Usage: identities <action> [args]",
		"Available actions:",
		"  " + usageListIdentities,
		"  " + usageNewIdentity,
		"  " + usageUnlockIdentity,
		"  " + usageRegisterIdentity,
		"  " + usageTopupIdentity,
		"  " + usageSettle,
	}, "\n")

	if len(argsString) == 0 {
		info(usage)
		return
	}

	args := strings.Fields(argsString)
	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "list":
		c.listIdentities(actionArgs)
	case "new":
		c.newIdentity(actionArgs)
	case "unlock":
		c.unlockIdentity(actionArgs)
	case "register":
		c.registerIdentity(actionArgs)
	case "topup":
		c.topupIdentity(actionArgs)
	case "settle":
		c.settle(actionArgs)
	default:
		warnf("Unknown sub-command '%s'\n", argsString)
		fmt.Println(usage)
	}
}

const usageListIdentities = "list"

func (c *cliApp) listIdentities(args []string) {
	if len(args) > 0 {
		info("Usage: " + usageListIdentities)
		return
	}
	ids, err := c.tequilapi.GetIdentities()
	if err != nil {
		fmt.Println("Error occurred:", err)
		return
	}

	for _, id := range ids {
		status("+", id.Address)
	}
	return
}

const usageNewIdentity = "new [passphrase]"

func (c *cliApp) newIdentity(args []string) {
	if len(args) > 1 {
		info("Usage: " + usageNewIdentity)
		return
	}
	passphrase := identityDefaultPassphrase
	if len(args) == 1 {
		passphrase = args[1]
	}

	id, err := c.tequilapi.NewIdentity(passphrase)
	if err != nil {
		warn(err)
		return
	}
	success("New identity created:", id.Address)
}

const usageUnlockIdentity = "unlock <identity> [passphrase]"

func (c *cliApp) unlockIdentity(actionArgs []string) {
	if len(actionArgs) < 1 {
		info("Usage: " + usageUnlockIdentity)
		return
	}

	identity := actionArgs[0]
	var passphrase string
	if len(actionArgs) >= 2 {
		passphrase = actionArgs[1]
	}

	info("Unlocking", identity)
	err := c.tequilapi.Unlock(identity, passphrase)
	if err != nil {
		warn(err)
		return
	}

	success(fmt.Sprintf("Identity %s unlocked.", identity))
}

const usageRegisterIdentity = "register <identity> [stake] [beneficiary]"

func (c *cliApp) registerIdentity(actionArgs []string) {
	if len(actionArgs) < 1 || len(actionArgs) > 3 {
		info("Usage: " + usageRegisterIdentity)
		return
	}

	var address = actionArgs[0]
	var stake uint64
	if len(actionArgs) >= 2 {
		s, err := strconv.ParseUint(actionArgs[1], 10, 64)
		if err != nil {
			warn(errors.Wrap(err, "could not parse stake"))
		}
		stake = s
	}
	var beneficiary string
	if len(actionArgs) >= 3 {
		beneficiary = actionArgs[2]
	}

	fees, err := c.tequilapi.GetTransactorFees()
	if err != nil {
		warn(err)
		return
	}

	err = c.tequilapi.RegisterIdentity(address, beneficiary, stake, fees.Registration)
	if err != nil {
		warn(errors.Wrap(err, "could not register identity"))
		return
	}
	info("Waiting for registration to complete")
	var registered, timeout bool
	for timer := time.After(3 * time.Minute); !timeout && !registered; {
		time.Sleep(2 * time.Second)
		status, err := c.tequilapi.GetIdentityStatus(address)
		if err != nil {
			warn(err)
		}
		select {
		case <-timer:
			timeout = true
		default:
			if status.IsRegistered {
				registered = true
				fmt.Println()
			} else {
				fmt.Print(".")
			}
		}
	}

	if registered {
		success("Identity registered")
	} else if timeout {
		warn("Identity registration timed out")
	} else {
		warn("Something went wrong")
	}
}

const usageTopupIdentity = "topup <identity>"

func (c *cliApp) topupIdentity(args []string) {
	if len(args) != 1 {
		info("Usage: " + usageTopupIdentity)
		return
	}

	address := args[0]
	err := c.tequilapi.TopUp(address)
	if err != nil {
		warn(err)
		return
	}
	success("Identity topped up")
}

const usageSettle = "settle <providerIdentity>"

func (c *cliApp) settle(args []string) {
	if len(args) != 1 {
		info("Usage: " + usageSettle)
		return
	}
	accountantID := config.GetString(config.FlagAccountantID)
	info("Waiting for settlement to complete")
	errChan := make(chan error)

	go func() {
		errChan <- c.tequilapi.Settle(identity.FromAddress(args[0]), identity.FromAddress(accountantID), true)
	}()

	timeout := time.After(time.Minute * 2)
	for {
		select {
		case <-timeout:
			fmt.Println()
			warn("Settlement timed out")
			return
		case <-time.After(time.Millisecond * 500):
			fmt.Print(".")
		case err := <-errChan:
			fmt.Println()
			if err != nil {
				warn("settlement failed: ", err.Error())
				return
			}
			info("settlement succeeded")
			return
		}
	}
}
