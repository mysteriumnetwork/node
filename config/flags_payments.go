/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package config

import (
	"time"

	"github.com/urfave/cli/v2"

	"github.com/mysteriumnetwork/node/metadata"
)

var (
	// FlagPaymentsMaxHermesFee represents the max hermes fee.
	FlagPaymentsMaxHermesFee = cli.IntFlag{
		Name:  "payments.hermes.max.fee",
		Value: 3000,
		Usage: "The max fee that we'll accept from an hermes. In percentiles. 3000 means 30%",
	}
	// FlagPaymentsBCTimeout represents the BC call timeout.
	FlagPaymentsBCTimeout = cli.DurationFlag{
		Name:  "payments.bc.timeout",
		Value: time.Second * 30,
		Usage: "The duration we'll wait before timing out BC calls.",
	}
	// FlagPaymentsHermesPromiseSettleThreshold represents the percentage of balance left when we go for promise settling.
	FlagPaymentsHermesPromiseSettleThreshold = cli.Float64Flag{
		Name:  "payments.hermes.promise.threshold",
		Value: 0.1,
		Usage: "The percentage of balance before we settle promises",
	}
	// FlagPaymentsHermesPromiseSettleTimeout represents the time we wait for confirmation of the promise settlement.
	FlagPaymentsHermesPromiseSettleTimeout = cli.DurationFlag{
		Name:   "payments.hermes.settle.timeout",
		Value:  time.Minute * 3,
		Usage:  "The duration we'll wait before timing out our wait for promise settle.",
		Hidden: true,
	}
	// FlagPaymentsHermesPromiseSettleCheckInterval represents the time for polling for confirmation of the promise settlement.
	FlagPaymentsHermesPromiseSettleCheckInterval = cli.DurationFlag{
		Name:   "payments.hermes.settle.check-interval",
		Value:  time.Second * 30,
		Usage:  "The duration we'll wait before trying to fetch new events.",
		Hidden: true,
	}
	// FlagPaymentsLongBalancePollInterval determines how often we resync balance on chain.
	FlagPaymentsLongBalancePollInterval = cli.DurationFlag{
		Name:   "payments.balance-long-poll.interval",
		Value:  time.Hour * 1,
		Usage:  "The duration we'll wait before trying to fetch new balance.",
		Hidden: true,
	}
	// FlagPaymentsFastBalancePollInterval determines how often we resync balance on chain after on chain events.
	FlagPaymentsFastBalancePollInterval = cli.DurationFlag{
		Name:   "payments.balance-short-poll.interval",
		Value:  time.Minute,
		Usage:  "The duration we'll wait before trying to fetch new balance.",
		Hidden: true,
	}
	// FlagPaymentsFastBalancePollTimeout determines how long we try to resync balance on chain after on chain events.
	FlagPaymentsFastBalancePollTimeout = cli.DurationFlag{
		Name:   "payments.balance-short-poll.timeout",
		Value:  time.Minute * 10,
		Usage:  "The duration we'll wait before giving up trying to fetch new balance.",
		Hidden: true,
	}
	// FlagPaymentsZeroStakeUnsettledAmount determines the minimum amount of myst that we will settle automatically if zero stake is used.
	FlagPaymentsZeroStakeUnsettledAmount = cli.Float64Flag{
		Name:  "payments.zero-stake-unsettled-amount",
		Value: 5.0,
		Usage: "The settling threshold if provider uses a zero stake",
	}
	// FlagPaymentsPromiseSettleMaxFeeThreshold represents the max percentage of the settlement that will be acceptable to pay in transaction fees.
	FlagPaymentsPromiseSettleMaxFeeThreshold = cli.Float64Flag{
		Name:  "payments.settle.max-fee-percentage",
		Value: 0.05,
		Usage: "The max percentage we allow to pay in fees when automatically settling promises.",
	}
	// FlagPaymentsUnsettledMaxAmount determines the maximum amount of myst for which we will consider the fee threshold.
	FlagPaymentsUnsettledMaxAmount = cli.Float64Flag{
		Name:  "payments.unsettled.max-amount",
		Value: 20.0,
		Usage: "The maximum amount of unsettled myst, after that we will always try to settle.",
	}
	// FlagPaymentsRegistryTransactorPollInterval The duration we'll wait before calling transactor to check for new status updates.
	FlagPaymentsRegistryTransactorPollInterval = cli.DurationFlag{
		Name:   "payments.registry-transactor-poll.interval",
		Value:  time.Second * 20,
		Usage:  "The duration we'll wait before calling transactor to check for new status updates",
		Hidden: true,
	}
	// FlagPaymentsRegistryTransactorPollTimeout The duration we'll wait before polling up the transactors registration status again.
	FlagPaymentsRegistryTransactorPollTimeout = cli.DurationFlag{
		Name:   "payments.registry-transactor-poll.timeout",
		Value:  time.Minute * 20,
		Usage:  "The duration we'll wait before giving up on transactors registration status",
		Hidden: true,
	}
	// FlagPaymentsConsumerDataLeewayMegabytes sets the data amount the consumer agrees to pay before establishing a session
	FlagPaymentsConsumerDataLeewayMegabytes = cli.Uint64Flag{
		Name:  metadata.FlagNames.PaymentsDataLeewayMegabytes,
		Usage: "sets the data amount the consumer agrees to pay before establishing a session",
		Value: metadata.MainnetDefinition.Payments.DataLeewayMegabytes,
	}
	// FlagPaymentsHermesStatusRecheckInterval sets how often we re-check the hermes status on bc. Higher values allow for less bc lookups but increase the risk for provider.
	FlagPaymentsHermesStatusRecheckInterval = cli.DurationFlag{
		Hidden: true,
		Name:   "payments.provider.hermes-status-recheck-interval",
		Usage:  "sets the hermes status recheck interval. Setting this to a lower value will decrease potential loss in case of Hermes getting locked.",
		Value:  time.Hour * 2,
	}
	// FlagOffchainBalanceExpiration sets how often we re-check offchain balance on hermes when balance is depleting
	FlagOffchainBalanceExpiration = cli.DurationFlag{
		Hidden: true,
		Name:   "payments.consumer.offchain-expiration",
		Usage:  "after syncing offchain balance, how long should node wait for next check to occur",
		Value:  time.Minute * 30,
	}
	// FlagPaymentsDuringSessionDebug sets if we're in debug more for the payments done in a VPN session.
	FlagPaymentsDuringSessionDebug = cli.BoolFlag{
		Name:   "payments.during-session-debug",
		Usage:  "Set debug mode for payments made during a session, it will bypass any price validation and allow absurd prices during sessions",
		Value:  false,
		Hidden: true,
	}
	// FlagPaymentsAmountDuringSessionDebug sets the amount of MYST sent during session debug
	FlagPaymentsAmountDuringSessionDebug = cli.Uint64Flag{
		Name:   "payments.amount-during-session-debug-amount",
		Usage:  "Set amount to pay during session debug",
		Value:  5000000000000000000,
		Hidden: true,
	}

	// FlagObserverAddress address of Observer service.
	FlagObserverAddress = cli.StringFlag{
		Name:  metadata.FlagNames.ObserverAddress,
		Usage: "full address of the observer service",
		Value: metadata.DefaultNetwork.ObserverAddress,
	}

	// FlagPaymentsLimitUnpaidInvoiceValue sets the upper limit of session payment value before forcing an invoice
	FlagPaymentsLimitUnpaidInvoiceValue = cli.StringFlag{
		Name:  "payments.provider.max-unpaid-invoice-value-limit",
		Usage: "sets the max upper limit of session payment value before forcing an invoice. If this value is exceeded before a payment interval is reached, an invoice is sent.",
		Value: "30000000000000000",
	}

	// FlagPaymentsUnpaidInvoiceValue sets the starting max limit of session payment value before forcing an invoice
	FlagPaymentsUnpaidInvoiceValue = cli.StringFlag{
		Name:   "payments.provider.max-unpaid-invoice-value",
		Usage:  "sets the starting upper limit of session payment value before forcing an invoice. If this value is exceeded before a payment interval is reached, an invoice is sent.",
		Value:  "3000000000000000",
		Hidden: true,
	}

	// FlagPaymentsProviderInvoiceFrequency determines how often the provider sends invoices.
	FlagPaymentsProviderInvoiceFrequency = cli.DurationFlag{
		Name:   "payments.provider.invoice-frequency",
		Value:  time.Second * 5,
		Usage:  "Determines how often the provider sends invoices.",
		Hidden: true,
	}

	// FlagPaymentsLimitProviderInvoiceFrequency determines how often the provider sends invoices.
	FlagPaymentsLimitProviderInvoiceFrequency = cli.DurationFlag{
		Name:  "payments.provider.invoice-frequency-limit",
		Value: time.Minute * 5,
		Usage: "Determines how often the provider sends invoices.",
	}
)

// RegisterFlagsPayments function register payments flags to flag list.
func RegisterFlagsPayments(flags *[]cli.Flag) {
	*flags = append(
		*flags,
		&FlagPaymentsMaxHermesFee,
		&FlagPaymentsBCTimeout,
		&FlagPaymentsHermesPromiseSettleThreshold,
		&FlagPaymentsPromiseSettleMaxFeeThreshold,
		&FlagPaymentsUnsettledMaxAmount,
		&FlagPaymentsHermesPromiseSettleTimeout,
		&FlagPaymentsHermesPromiseSettleCheckInterval,
		&FlagPaymentsLongBalancePollInterval,
		&FlagPaymentsFastBalancePollInterval,
		&FlagPaymentsFastBalancePollTimeout,
		&FlagPaymentsRegistryTransactorPollTimeout,
		&FlagPaymentsRegistryTransactorPollInterval,
		&FlagPaymentsConsumerDataLeewayMegabytes,
		&FlagPaymentsHermesStatusRecheckInterval,
		&FlagOffchainBalanceExpiration,
		&FlagPaymentsZeroStakeUnsettledAmount,
		&FlagPaymentsDuringSessionDebug,
		&FlagPaymentsAmountDuringSessionDebug,
		&FlagObserverAddress,

		&FlagPaymentsProviderInvoiceFrequency,
		&FlagPaymentsLimitProviderInvoiceFrequency,

		&FlagPaymentsUnpaidInvoiceValue,
		&FlagPaymentsLimitUnpaidInvoiceValue,
	)
}

// ParseFlagsPayments function fills in payments options from CLI context.
func ParseFlagsPayments(ctx *cli.Context) {
	Current.ParseIntFlag(ctx, FlagPaymentsMaxHermesFee)
	Current.ParseDurationFlag(ctx, FlagPaymentsBCTimeout)
	Current.ParseFloat64Flag(ctx, FlagPaymentsHermesPromiseSettleThreshold)
	Current.ParseFloat64Flag(ctx, FlagPaymentsPromiseSettleMaxFeeThreshold)
	Current.ParseFloat64Flag(ctx, FlagPaymentsUnsettledMaxAmount)
	Current.ParseDurationFlag(ctx, FlagPaymentsHermesPromiseSettleTimeout)
	Current.ParseDurationFlag(ctx, FlagPaymentsHermesPromiseSettleCheckInterval)
	Current.ParseDurationFlag(ctx, FlagPaymentsFastBalancePollInterval)
	Current.ParseDurationFlag(ctx, FlagPaymentsFastBalancePollTimeout)
	Current.ParseDurationFlag(ctx, FlagPaymentsLongBalancePollInterval)
	Current.ParseDurationFlag(ctx, FlagPaymentsLongBalancePollInterval)
	Current.ParseDurationFlag(ctx, FlagPaymentsRegistryTransactorPollInterval)
	Current.ParseDurationFlag(ctx, FlagPaymentsRegistryTransactorPollTimeout)
	Current.ParseUInt64Flag(ctx, FlagPaymentsConsumerDataLeewayMegabytes)
	Current.ParseDurationFlag(ctx, FlagPaymentsHermesStatusRecheckInterval)
	Current.ParseDurationFlag(ctx, FlagOffchainBalanceExpiration)
	Current.ParseFloat64Flag(ctx, FlagPaymentsZeroStakeUnsettledAmount)
	Current.ParseBoolFlag(ctx, FlagPaymentsDuringSessionDebug)
	Current.ParseUInt64Flag(ctx, FlagPaymentsAmountDuringSessionDebug)
	Current.ParseStringFlag(ctx, FlagObserverAddress)

	Current.ParseDurationFlag(ctx, FlagPaymentsProviderInvoiceFrequency)
	Current.ParseDurationFlag(ctx, FlagPaymentsLimitProviderInvoiceFrequency)

	Current.ParseStringFlag(ctx, FlagPaymentsLimitUnpaidInvoiceValue)
	Current.ParseStringFlag(ctx, FlagPaymentsUnpaidInvoiceValue)
}
