/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/cmd/commands/cli/clio"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/core/beneficiary"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/money"
	"github.com/mysteriumnetwork/payments/crypto"
)

func (c *cliApp) identities(args []string) (err error) {
	usage := strings.Join([]string{
		"Usage: identities <action> [args]",
		"Available actions:",
		"  " + usageListIdentities,
		"  " + usageGetIdentity,
		"  " + usageGetBalance,
		"  " + usageNewIdentity,
		"  " + usageUnlockIdentity,
		"  " + usageRegisterIdentity,
		"  " + usageSettle,
		"  " + usageSetPayoutAddress,
		"  " + usageGetPayoutAddress,
		"  " + usageSetBeneficiary,
		"  " + usageSetBeneficiaryStatus,
		"  " + usageGetReferralCode,
		"  " + usageExportIdentity,
		"  " + usageImportIdentity,
		"  " + usageWithdraw,
		"  " + usageLastWithdrawal,
		"  " + usageMigrateHermesStatus,
		"  " + usageMigrateHermes,
	}, "\n")

	if len(args) == 0 {
		clio.Info(usage)
		return
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "list":
		return c.listIdentities(actionArgs)
	case "get":
		return c.getIdentity(actionArgs)
	case "balance":
		return c.getBalance(actionArgs)
	case "new":
		return c.newIdentity(actionArgs)
	case "unlock":
		return c.unlockIdentity(actionArgs)
	case "register":
		return c.registerIdentity(actionArgs)
	case "settle":
		return c.settle(actionArgs)
	case "set-payout-address":
		return c.setPayoutAddress(actionArgs)
	case "get-payout-address":
		return c.getPayoutAddress(actionArgs)
	case "beneficiary-set":
		return c.setBeneficiary(actionArgs)
	case "beneficiary-status":
		return c.getBeneficiaryStatus(actionArgs)
	case "referralcode":
		return c.getReferralCode(actionArgs)
	case "export":
		return c.exportIdentity(actionArgs)
	case "import":
		return c.importIdentity(actionArgs)
	case "withdraw":
		return c.withdraw(actionArgs)
	case "last-withdrawal":
		return c.lastWithdrawal(actionArgs)
	case "migrate-hermes":
		return c.migrateHermes(actionArgs)
	case "migrate-hermes-status":
		return c.migrateHermesStatus(actionArgs)
	default:
		fmt.Println(usage)
		return errUnknownSubCommand(args[0])
	}
}

const usageListIdentities = "list"

func (c *cliApp) listIdentities(args []string) (err error) {
	if len(args) > 0 {
		clio.Info("Usage: " + usageListIdentities)
		return errWrongArgumentCount
	}
	ids, err := c.tequilapi.GetIdentities()
	if err != nil {
		return err
	}

	for _, id := range ids {
		clio.Status("+", id.Address)
	}
	return nil
}

const usageGetBalance = "balance <identity>"

func (c *cliApp) getBalance(actionArgs []string) (err error) {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageGetBalance)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	balance, err := c.tequilapi.BalanceRefresh(address)
	if err != nil {
		return err
	}

	clio.Info(fmt.Sprintf("Balance: %s MYST", balance.BalanceTokens))
	return nil
}

const usageGetIdentity = "get <identity>"

func (c *cliApp) getIdentity(actionArgs []string) (err error) {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageGetIdentity)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	identityStatus, err := c.tequilapi.Identity(address)
	if err != nil {
		return err
	}
	clio.Info("Registration Status: ", identityStatus.RegistrationStatus)
	clio.Info("Channel address: ", identityStatus.ChannelAddress)
	clio.Info(fmt.Sprintf("Balance: %s MYST", identityStatus.BalanceTokens))
	clio.Info(fmt.Sprintf("Earnings: %s", money.New(identityStatus.Earnings)))
	clio.Info(fmt.Sprintf("Earnings total: %s", money.New(identityStatus.EarningsTotal)))
	return nil
}

const usageNewIdentity = "new [passphrase]"

func (c *cliApp) newIdentity(args []string) (err error) {
	if len(args) > 1 {
		clio.Info("Usage: " + usageNewIdentity)
		return errWrongArgumentCount
	}
	passphrase := identityDefaultPassphrase
	if len(args) == 1 {
		passphrase = args[0]
	}

	id, err := c.tequilapi.NewIdentity(passphrase)
	if err != nil {
		return err
	}
	clio.Success("New identity created:", id.Address)
	return nil
}

const usageUnlockIdentity = "unlock <identity> [passphrase]"

func (c *cliApp) unlockIdentity(actionArgs []string) (err error) {
	if len(actionArgs) < 1 {
		clio.Info("Usage: " + usageUnlockIdentity)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	var passphrase string
	if len(actionArgs) >= 2 {
		passphrase = actionArgs[1]
	}

	clio.Info("Unlocking ", address)
	err = c.tequilapi.Unlock(address, passphrase)
	if err != nil {
		return err
	}

	clio.Success(fmt.Sprintf("Identity %s unlocked.", address))
	return nil
}

const usageRegisterIdentity = "register <identity> [beneficiary] [referralcode]"

func (c *cliApp) registerIdentity(actionArgs []string) error {
	if len(actionArgs) < 1 || len(actionArgs) > 3 {
		clio.Info("Usage: " + usageRegisterIdentity)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	var beneficiary string
	if len(actionArgs) >= 2 {
		beneficiary = actionArgs[1]
	}

	var token *string
	if len(actionArgs) >= 3 {
		token = &actionArgs[2]
	}

	err := c.tequilapi.RegisterIdentity(address, beneficiary, token)
	if err != nil {
		return fmt.Errorf("could not register identity: %w", err)
	}

	msg := "Registration started. Top up the identities channel to finish it."

	clio.Info(msg)
	clio.Info(fmt.Sprintf("To explore additional information about the identity use: identities %s", usageGetIdentity))
	return nil
}

const usageSettle = "settle <providerIdentity> [hermesID,hermesID2]"

func (c *cliApp) settle(args []string) (err error) {
	if len(args) == 0 || len(args) > 2 {
		clio.Info("Usage: " + usageSettle)
		fees, err := c.tequilapi.GetTransactorFees()
		if err != nil {
			clio.Warn("Could not get transactor fee:", err)
		}
		clio.Info(fmt.Sprintf("Transactor fee: %s MYST", fees.SettlementTokens))

		hermesPct, err := decimal.NewFromString(fees.HermesPercent)
		if err != nil {
			clio.Info("Could not retrieve hermes fee:", err)
		}
		clio.Info(fmt.Sprintf("Hermes fee: %s%%", hermesPct.Mul(decimal.NewFromInt(100)).StringFixed(2)))
		return errWrongArgumentCount
	}

	hermesID, err := c.config.GetHermesID()
	if err != nil {
		return fmt.Errorf("could not get Hermes ID: %w", err)
	}

	hermesIDs := make([]common.Address, 0)
	if len(args) == 2 {
		ids := strings.Split(args[1], ",")
		for _, h := range ids {
			hermesIDs = append(hermesIDs, common.HexToAddress(h))
		}
	} else {
		hermesIDs = append(hermesIDs, common.HexToAddress(hermesID))
	}

	clio.Infof("Will settle with hermes: %v\n", hermesIDs)
	clio.Info("Waiting for settlement to complete")
	errChan := make(chan error)

	go func() {
		errChan <- c.tequilapi.Settle(identity.FromAddress(args[0]), hermesIDs, true)
	}()

	timeout := time.After(time.Minute * 5)
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

const usageGetPayoutAddress = "get-payout-address <identity>"

func (c *cliApp) getPayoutAddress(args []string) error {
	if len(args) != 1 {
		clio.Info("Usage: " + usageGetPayoutAddress)
		return errWrongArgumentCount
	}
	addr, err := c.tequilapi.GetPayout(args[0])
	if err != nil {
		return fmt.Errorf("could not get payout address: %w", err)
	}
	clio.Info("Payout address: ", addr.Address)
	return nil
}

const usageSetPayoutAddress = "set-payout-address <providerIdentity> <beneficiary>"

func (c *cliApp) setPayoutAddress(args []string) error {
	if len(args) != 2 {
		clio.Info("Usage: " + usageSetPayoutAddress)
		return errWrongArgumentCount
	}
	err := c.tequilapi.SetPayout(args[0], args[1])
	if err != nil {
		return fmt.Errorf("could not set payout address: %w", err)
	}

	clio.Info("Payout address set to: ", args[0])
	return nil
}

const usageSetBeneficiary = "beneficiary-set <providerIdentity> <beneficiary> [hermesID]"

func (c *cliApp) setBeneficiary(actionArgs []string) error {
	if len(actionArgs) < 2 || len(actionArgs) > 3 {
		clio.Info("Usage: " + usageSetBeneficiary)
		return errors.New("malformed args")
	}

	address := actionArgs[0]
	benef := actionArgs[1]
	hermesID := ""
	if len(actionArgs) == 3 {
		hermesID = actionArgs[2]
	}

	err := c.tequilapi.SettleWithBeneficiary(address, benef, hermesID)
	if err != nil {
		return err
	}

	timeout := time.After(30 * time.Second)
	for {
		select {
		case <-timeout:
			clio.Info("Beneficiary change in progress")
			clio.Info(fmt.Sprintf("To get additional information use command: \"%s\"", usageSetBeneficiaryStatus))
			return nil
		case <-time.After(time.Second):
			st, err := c.tequilapi.SettleWithBeneficiaryStatus(address)
			if err != nil {
				break
			}

			if !strings.EqualFold(st.ChangeTo, benef) || st.State != beneficiary.Completed {
				break
			}

			if st.Error != "" {
				return fmt.Errorf("could not set new beneficiary address: %s", st.Error)
			}

			data, err := c.tequilapi.Beneficiary(address)
			if err != nil {
				break
			}

			if strings.EqualFold(data.Beneficiary, benef) {
				clio.Success("New beneficiary address set")
				return nil
			}
		}
	}
}

const usageSetBeneficiaryStatus = "beneficiary-status <identity>"

func (c *cliApp) getBeneficiaryStatus(actionArgs []string) error {
	if len(actionArgs) != 1 {
		return errors.New("malformed args")
	}

	address := actionArgs[0]

	data, err := c.tequilapi.Beneficiary(address)
	if err != nil {
		return fmt.Errorf("could not get current beneficiary: %w", err)
	}

	clio.Info(fmt.Sprintf("Current beneficiary: %s", data.Beneficiary))
	clio.Info(fmt.Sprintf("Is a channel address: %t", data.IsChannelAddress))

	st, err := c.tequilapi.SettleWithBeneficiaryStatus(address)
	if err != nil {
		return fmt.Errorf("could not get beneficiary change status: %w", err)
	}

	clio.Info("Last change request information:")
	clio.Info(fmt.Sprintf("Change to: %s", st.ChangeTo))
	clio.Info(fmt.Sprintf("State: %s", st.State))
	if st.Error != "" {
		clio.Warn(fmt.Sprintf("Error: %s", st.Error))
	}

	return nil
}

const usageWithdraw = "withdraw <providerIdentity> <beneficiary> <toChain> [amount]"

func (c *cliApp) withdraw(args []string) error {
	if len(args) < 3 {
		clio.Info("Usage: " + usageWithdraw)
		fees, err := c.tequilapi.GetTransactorFees()
		if err != nil {
			clio.Warn("Could not get transactor fee:", err)
		}
		clio.Info(fmt.Sprintf("Transactor fee: %s MYST", fees.SettlementTokens))
		hermesPct, err := decimal.NewFromString(fees.HermesPercent)
		if err != nil {
			clio.Info("Could not retrieve hermes fee:", err)
		}
		clio.Info(fmt.Sprintf("Hermes fee: %s%%", hermesPct.Mul(decimal.NewFromInt(100)).StringFixed(2)))
		return errWrongArgumentCount
	}
	hermesID, err := c.config.GetHermesID()
	if err != nil {
		return fmt.Errorf("could not get Hermes ID: %w", err)
	}
	errChan := make(chan error)

	fromChain := c.config.GetInt64ByFlag(config.FlagChain2ChainID)

	providerIdentity := args[0]
	if !common.IsHexAddress(providerIdentity) {
		return errors.New("a valid provider identity must be provided")
	}

	beneficiaryAddr := args[1]
	if !common.IsHexAddress(beneficiaryAddr) {
		return errors.New("a valid beneficiary address must be provided")
	}

	toChain, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("could not parse chain id %w", err)
	}

	var amount *big.Int
	if len(args) == 4 {
		amf, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			return fmt.Errorf("%v is not a valid number", args[3])
		}
		if amf > 99 {
			return errors.New("max withdrawal amount is 99 MYST")
		}

		amount = crypto.FloatToBigMyst(amf)
	}

	clio.Info("Waiting for withdrawal to complete")
	go func() {
		errChan <- c.tequilapi.Withdraw(identity.FromAddress(providerIdentity), common.HexToAddress(hermesID), common.HexToAddress(beneficiaryAddr), amount, fromChain, int64(toChain))
	}()

	timeout := time.After(time.Minute * 2)
	for {
		select {
		case <-timeout:
			return errors.New("withdrawal timed out")
		case <-time.After(time.Millisecond * 500):
			fmt.Print(".")
		case err := <-errChan:
			fmt.Println()
			if err != nil {
				return fmt.Errorf("withdrawal failed: %w", err)
			}
			clio.Info("withdrawal succeeded")
			return nil
		}
	}
}

const usageGetReferralCode = "referralcode <identity>"

func (c *cliApp) getReferralCode(actionArgs []string) error {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageGetReferralCode)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	res, err := c.tequilapi.IdentityReferralCode(address)
	if err != nil {
		return fmt.Errorf("could not get referral token: %w", err)
	}

	clio.Success(fmt.Sprintf("Your referral token is: %q", res.Token))
	return nil
}

const usageExportIdentity = "export <identity> <new_passphrase> [file]"

func (c *cliApp) exportIdentity(actionsArgs []string) (err error) {
	if len(actionsArgs) < 2 || len(actionsArgs) > 3 {
		clio.Info("Usage: " + usageExportIdentity)
		return errWrongArgumentCount
	}

	id := actionsArgs[0]
	passphrase := actionsArgs[1]

	dataDir := c.config.GetStringByFlag(config.FlagDataDir)
	if dataDir == "" {
		return errors.New("could not get data directory")
	}

	ksdir := node.GetOptionsDirectoryKeystore(dataDir)
	ks := keystore.NewKeyStore(ksdir, keystore.LightScryptN, keystore.LightScryptP)

	ex := identity.NewExporter(identity.NewKeystoreFilesystem(ksdir, ks))

	blob, err := ex.Export(id, "", passphrase)
	if err != nil {
		return fmt.Errorf("failed to export identity: %w", err)
	}

	if len(actionsArgs) == 3 {
		filepath := actionsArgs[2]
		write := func() error {
			f, err := os.Create(filepath)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.Write(blob)
			return err
		}

		err := write()
		if err != nil {
			return fmt.Errorf("failed to write exported key to file: %s reason: %w", filepath, err)
		}

		clio.Success("Identity exported to file:", filepath)
		return nil
	}

	clio.Success("Private key exported: ")
	fmt.Println(string(blob))
	return nil
}

const usageImportIdentity = "import <passphrase> <key-string/key-file>"

func (c *cliApp) importIdentity(actionsArgs []string) (err error) {
	if len(actionsArgs) != 2 {
		clio.Info("Usage: " + usageImportIdentity)
		return errWrongArgumentCount
	}

	key := actionsArgs[1]
	passphrase := actionsArgs[0]

	blob := []byte(key)
	if _, err := os.Stat(key); err == nil {
		blob, err = ioutil.ReadFile(key)
		if err != nil {
			return fmt.Errorf("can't read provided file: %s reason: %w", key, err)
		}
	}

	id, err := c.tequilapi.ImportIdentity(blob, passphrase, true)
	if err != nil {
		return fmt.Errorf("failed to import identity: %w", err)
	}

	clio.Success("Identity imported:", id.Address)
	return nil
}

const usageLastWithdrawal = "last-withdrawal <identity>"

func (c *cliApp) lastWithdrawal(actionArgs []string) error {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageLastWithdrawal)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	history, err := c.tequilapi.WithdrawalHistory(address)
	if err != nil {
		return fmt.Errorf("could not get last withdrawal: %w", err)
	}
	if history.TotalItems == 0 {
		clio.Info("No withdrawals found")
		return nil
	}

	lastWithdrawal := history.Items[0]

	clio.Info("Tx hash: ", lastWithdrawal.TxHash)
	clio.Info("DateTime: ", lastWithdrawal.SettledAt)
	clio.Info("Beneficiary: ", lastWithdrawal.Beneficiary)
	clio.Info("Amount: ", money.New(lastWithdrawal.Amount))
	clio.Info("Fees: ", money.New(lastWithdrawal.Fees))
	clio.Info("Blockchain explorer URL: ", lastWithdrawal.BlockExplorerURL)
	if lastWithdrawal.Error != "" {
		clio.Warnf("Error: %s", lastWithdrawal.Error)
	} else {
		clio.Info("Error: none")
	}
	return nil
}

const usageMigrateHermesStatus = "migrate-hermes-status <identity>"

func (c *cliApp) migrateHermesStatus(actionArgs []string) error {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageMigrateHermesStatus)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	response, err := c.tequilapi.MigrateHermesStatus(address)
	if err != nil {
		return err
	}

	clio.Info("Migration status: ", response.Status)

	return nil
}

const usageMigrateHermes = "migrate-hermes <identity>"

func (c *cliApp) migrateHermes(actionArgs []string) error {
	if len(actionArgs) != 1 {
		clio.Info("Usage: " + usageMigrateHermes)
		return errWrongArgumentCount
	}

	address := actionArgs[0]
	clio.Info("Migration started")
	err := c.tequilapi.MigrateHermes(address)
	if err != nil {
		return err
	}
	clio.Info("Migration finished successfully")

	return nil
}
