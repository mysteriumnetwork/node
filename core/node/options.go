/*
 * Copyright (C) 2018 The "MysteriumNetwork/node" Authors.
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

package node

import (
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/logconfig"
	openvpn_core "github.com/mysteriumnetwork/node/services/openvpn/core"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Openvpn interface is abstraction over real openvpn options to unblock mobile development
// will disappear as soon as go-openvpn will unify common factory for openvpn creation
type Openvpn interface {
	Check() error
	BinaryPath() string
}

// TODO this struct will disappear when we unify go-openvpn embedded lib and external process based session creation/handling
type wrapper struct {
	nodeOptions openvpn_core.NodeOptions
}

func (w wrapper) Check() error {
	return w.nodeOptions.Check()
}

func (w wrapper) BinaryPath() string {
	return w.nodeOptions.BinaryPath
}

var _ Openvpn = wrapper{}

// Options describes options which are required to start Node
type Options struct {
	Directories OptionsDirectory

	TequilapiAddress string
	TequilapiPort    int
	TequilapiEnabled bool
	BindAddress      string
	UI               OptionsUI
	FeedbackURL      string

	Keystore OptionsKeystore

	logconfig.LogOptions
	OptionsNetwork
	Discovery  OptionsDiscovery
	MMN        OptionsMMN
	Quality    OptionsQuality
	Location   OptionsLocation
	Transactor OptionsTransactor
	Accountant OptionsAccountant

	Openvpn  Openvpn
	Firewall OptionsFirewall

	Payments OptionsPayments
}

// GetOptions retrieves node options from the app configuration.
func GetOptions() *Options {
	return &Options{
		Directories:      *GetOptionsDirectory(),
		TequilapiAddress: config.GetString(config.FlagTequilapiAddress),
		TequilapiPort:    config.GetInt(config.FlagTequilapiPort),
		TequilapiEnabled: true,
		BindAddress:      config.GetString(config.FlagBindAddress),
		UI: OptionsUI{
			UIEnabled: config.GetTBool(config.FlagUIEnable),
			UIPort:    config.GetInt(config.FlagUIPort),
		},
		FeedbackURL: config.GetString(config.FlagFeedbackURL),
		Keystore: OptionsKeystore{
			UseLightweight: config.GetBool(config.FlagKeystoreLightweight),
		},
		LogOptions: *GetLogOptions(),
		OptionsNetwork: OptionsNetwork{
			Testnet:                     config.GetBool(config.FlagTestnet),
			Localnet:                    config.GetBool(config.FlagLocalnet),
			ExperimentNATPunching:       config.GetTBool(config.FlagNATPunching),
			MysteriumAPIAddress:         config.GetString(config.FlagAPIAddress),
			AccessPolicyEndpointAddress: config.GetString(config.FlagAccessPolicyAddress),
			BrokerAddress:               config.GetString(config.FlagBrokerAddress),
			EtherClientRPC:              config.GetString(config.FlagEtherRPC),
			QualityOracle:               config.GetString(config.FlagQualityOracleAddress),
		},
		Discovery: OptionsDiscovery{
			Type:                   DiscoveryType(config.GetString(config.FlagDiscoveryType)),
			Address:                config.GetString(config.FlagDiscoveryAddress),
			ProposalFetcherEnabled: true,
		},
		MMN: OptionsMMN{
			Address: config.GetString(config.FlagMMNAddress),
			Enabled: config.GetTBool(config.FlagMMNEnabled),
		},
		Quality: OptionsQuality{
			Type:    QualityType(config.GetString(config.FlagQualityType)),
			Address: config.GetString(config.FlagQualityAddress),
		},
		Location: OptionsLocation{
			IPDetectorURL: config.GetString(config.FlagIPDetectorURL),
			Type:          LocationType(config.GetString(config.FlagLocationType)),
			Address:       config.GetString(config.FlagLocationAddress),
			Country:       config.GetString(config.FlagLocationCountry),
			City:          config.GetString(config.FlagLocationCity),
			NodeType:      config.GetString(config.FlagLocationNodeType),
		},
		Transactor: OptionsTransactor{
			TransactorEndpointAddress:       config.GetString(config.FlagTransactorAddress),
			RegistryAddress:                 config.GetString(config.FlagTransactorRegistryAddress),
			ChannelImplementation:           config.GetString(config.FlagTransactorChannelImplementation),
			ProviderMaxRegistrationAttempts: config.GetInt(config.FlagTransactorProviderMaxRegistrationAttempts),
			ProviderRegistrationRetryDelay:  config.GetDuration(config.FlagTransactorProviderRegistrationRetryDelay),
			ProviderRegistrationStake:       config.GetUInt64(config.FlagTransactorProviderRegistrationStake),
		},
		Payments: OptionsPayments{
			MaxAllowedPaymentPercentile:        config.GetInt(config.FlagPaymentsMaxAccountantFee),
			BCTimeout:                          config.GetDuration(config.FlagPaymentsBCTimeout),
			AccountantPromiseSettlingThreshold: config.GetFloat64(config.FlagPaymentsAccountantPromiseSettleThreshold),
			SettlementTimeout:                  config.GetDuration(config.FlagPaymentsAccountantPromiseSettleTimeout),
			MystSCAddress:                      config.GetString(config.FlagPaymentsMystSCAddress),
		},
		Accountant: OptionsAccountant{
			AccountantID:              config.GetString(config.FlagAccountantID),
			AccountantEndpointAddress: config.GetString(config.FlagAccountantAddress),
		},
		Openvpn: wrapper{nodeOptions: openvpn_core.NodeOptions{
			BinaryPath: config.GetString(config.FlagOpenvpnBinary),
		}},
		Firewall: OptionsFirewall{
			BlockAlways: config.GetBool(config.FlagFirewallKillSwitch),
		},
	}
}

// GetLogOptions retrieves logger options from the app configuration.
func GetLogOptions() *logconfig.LogOptions {
	logDir := config.GetString(config.FlagLogDir)
	level, err := zerolog.ParseLevel(config.GetString(config.FlagLogLevel))
	if err != nil {
		log.Error().Err(err).Msg("Failed to parse logging level")
		level = zerolog.DebugLevel
	}
	return &logconfig.LogOptions{
		LogLevel: level,
		LogHTTP:  config.GetBool(config.FlagLogHTTP),
		Filepath: logDir,
	}
}

// OptionsKeystore stores the keystore configuration
type OptionsKeystore struct {
	UseLightweight bool
}
