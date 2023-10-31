/*
 * Copyright (C) 2022 The "MysteriumNetwork/node" Authors.
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

package contract

// Err codes returned from TequilAPI.
// Once created, do not change the string value, because consumers may depend on it - it's part of the contract.
const (

	// Identity

	ErrCodeIDImport                      = "err_id_import"
	ErrCodeIDSetDefault                  = "err_id_set_default"
	ErrCodeIDUseOrCreate                 = "err_to_id_use_or_create"
	ErrCodeIDUnlock                      = "err_id_unlock"
	ErrCodeIDLocked                      = "err_id_locked"
	ErrCodeIDNotRegistered               = "err_id_not_registered"
	ErrCodeIDStatusUnknown               = "err_id_status_unknown"
	ErrCodeIDCreate                      = "err_id_create"
	ErrCodeIDRegistrationCheck           = "err_id_registration_status_check"
	ErrCodeIDBlockchainRegistrationCheck = "err_id_registration_blockchain_status_check"
	ErrCodeIDRegistrationInProgress      = "err_id_registration_in_progress"
	ErrCodeIDCalculateAddress            = "err_id_calculate_address"
	ErrCodeIDSaveBeneficiaryAddress      = "err_id_save_beneficiary_invalid_address"
	ErrCodeIDGetBeneficiaryAddress       = "err_id_get_beneficiary_address"
	ErrCodeHermesMigration               = "err_id_check_hermes_migration"
	ErrCodeCheckHermesMigrationStatus    = "err_id_check_hermes_migration_status"

	// Payment

	ErrCodePaymentCreate         = "err_payment_create"
	ErrCodePaymentGet            = "err_payment_get"
	ErrCodePaymentGetInvoice     = "err_payment_get_invoice"
	ErrCodePaymentList           = "err_payment_list"
	ErrCodePaymentListCurrencies = "err_payment_list_currencies"
	ErrCodePaymentGetOptions     = "err_payment_get_order_options"
	ErrCodePaymentListGateways   = "err_payment_list_gateways"

	// Referral

	ErrCodeReferralGetToken = "err_referral_get_token"
	ErrCodeBeneficiaryGet   = "err_beneficiary_get"

	// Config

	ErrCodeConfigSave = "err_config_save"

	// Connection

	ErrCodeConnectionAlreadyExists = "err_connection_already_exists"
	ErrCodeConnectionCancelled     = "err_connection_cancelled"
	ErrCodeConnect                 = "err_connect"
	ErrCodeNoConnectionExists      = "err_no_connection_exists"
	ErrCodeDisconnect              = "err_disconnect"

	// Feedback

	ErrCodeFeedbackSubmit = "err_feedback_submit"

	// MMN

	ErrCodeMMNNodeAlreadyClaimed      = "err_mmn_node_already_claimed"
	ErrCodeMMNAPIKey                  = "err_mmn_api_key"
	ErrCodeMMNRegistration            = "err_mmn_registration"
	ErrCodeMMNClaimRedirectURLMissing = "err_mmn_claim_redirect_url_missing"
	ErrCodeMMNClaimLink               = "err_mmn_claim_link"

	// NAT

	ErrCodeNATProbe = "err_nat_probe"

	// Proposals

	ErrCodeProposalsQuery          = "err_proposals_query"
	ErrCodeProposalsCountryQuery   = "err_proposals_countries_query"
	ErrCodeProposalsDetectLocation = "err_proposals_detect_location"
	ErrCodeProposalsPrices         = "err_proposals_prices"
	ErrCodeProposalsPresets        = "err_proposals_presets"
	ErrCodeProposalsServiceType    = "err_proposals_service_type"

	// Service

	ErrCodeServiceList     = "err_service_list"
	ErrCodeServiceGet      = "err_service_get"
	ErrCodeServiceRunning  = "err_service_running"
	ErrCodeServiceLocation = "err_service_location"
	ErrCodeServiceStart    = "err_service_start"
	ErrCodeServiceStop     = "err_service_stop"

	// Sessions

	ErrCodeSessionList         = "err_session_list"
	ErrCodeSessionListPaginate = "err_session_list_paginate"
	ErrCodeSessionStats        = "err_session_stats"
	ErrCodeSessionStatsDaily   = "err_session_stats_daily"

	// Transactor

	ErrCodeTransactorRegistration          = "err_transactor_registration"
	ErrCodeTransactorFetchFees             = "err_transactor_fetch_fees"
	ErrCodeTransactorDecreaseStake         = "err_transactor_decrease_stake"
	ErrCodeTransactorSettleHistory         = "err_transactor_settle_history"
	ErrCodeTransactorSettleHistoryPaginate = "err_transactor_settle_history_paginate"
	ErrCodeTransactorWithdraw              = "err_transactor_withdraw"
	ErrCodeTransactorSettle                = "err_transactor_settle_into_stake"
	ErrCodeTransactorSettleAsync           = "err_transactor_settle_into_stake_async"
	ErrCodeTransactorNoReward              = "err_transactor_no_reward"
	ErrCodeTransactorBeneficiary           = "err_transactor_beneficiary"
	ErrCodeTransactorBeneficiaryTxStatus   = "err_transactor_beneficiary_tx_status"

	// Affiliator

	ErrCodeAffiliatorNoReward = "err_affiliator_no_reward"
	ErrCodeAffiliatorFailed   = "err_affiliator_failed"

	// Other

	ErrCodeActiveHermes                    = "err_get_active_hermes"
	ErrCodeHermesFee                       = "err_hermes_fee"
	ErrCodeHermesSettle                    = "err_hermes_settle"
	ErrCodeHermesSettleAsync               = "err_hermes_settle_async"
	ErrCodeUILocalVersions                 = "err_ui_local_versions"
	ErrCodeUISwitchVersion                 = "err_ui_switch_version"
	ErrCodeUIDownload                      = "err_ui_download"
	ErrCodeUIBundledVersion                = "err_ui_bundled_version"
	ErrCodeUIUsedVersion                   = "err_ui_used_version"
	ErrorCodeProviderSessions              = "err_provider_sessions"
	ErrorCodeProviderTransferredData       = "err_provider_transferred_data"
	ErrorCodeProviderSessionsCount         = "err_provider_sessions_count"
	ErrorCodeProviderConsumersCount        = "err_provider_consumers_count"
	ErrorCodeProviderEarningsSeries        = "err_provider_earnings_series"
	ErrorCodeProviderSessionsSeries        = "err_provider_sessions_series"
	ErrorCodeProviderTransferredDataSeries = "err_provider_transferred_data_series"
	ErrorCodeProviderQuality               = "err_provider_quality"
	ErrorCodeProviderActivityStats         = "err_provider_activity_stats"
	ErrorCodeLatestReleaseInformation      = "err_latest_release_information"
	ErrorCodeProviderServiceEarnings       = "err_provider_service_earnings"
)
