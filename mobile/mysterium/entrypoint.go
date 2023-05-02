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

package mysterium

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/config"
	"github.com/mysteriumnetwork/node/consumer/entertainment"
	"github.com/mysteriumnetwork/node/consumer/migration"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/core/quality"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/core/service/servicestate"
	"github.com/mysteriumnetwork/node/core/state"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/registry"
	"github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/metadata"
	natprobe "github.com/mysteriumnetwork/node/nat/behavior"
	"github.com/mysteriumnetwork/node/pilvytis"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/node/router"
	"github.com/mysteriumnetwork/node/services/wireguard"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	pingpongEvent "github.com/mysteriumnetwork/node/session/pingpong/event"
	paymentClient "github.com/mysteriumnetwork/payments/client"
	"github.com/mysteriumnetwork/payments/crypto"
	"github.com/mysteriumnetwork/payments/units"

	"github.com/rs/zerolog"
)

// MobileNode represents node object tuned for mobile device.
type MobileNode struct {
	shutdown                  func() error
	node                      *cmd.Node
	stateKeeper               *state.Keeper
	connectionManager         connection.MultiManager
	locationResolver          *location.Cache
	natProber                 natprobe.NATProber
	identitySelector          selector.Handler
	signerFactory             identity.SignerFactory
	identityMover             *identity.Mover
	ipResolver                ip.Resolver
	eventBus                  eventbus.EventBus
	connectionRegistry        *connection.Registry
	proposalsManager          *proposalsManager
	feedbackReporter          *feedback.Reporter
	transactor                *registry.Transactor
	affiliator                *registry.Affiliator
	identityRegistry          registry.IdentityRegistry
	identityChannelCalculator *paymentClient.MultiChainAddressProvider
	consumerBalanceTracker    *pingpong.ConsumerBalanceTracker
	pilvytis                  *pilvytis.API
	pilvytisOrderIssuer       *pilvytis.OrderIssuer
	chainID                   int64
	startTime                 time.Time
	sessionStorage            SessionStorage
	entertainmentEstimator    *entertainment.Estimator
	residentCountry           *identity.ResidentCountry
	filterPresetStorage       *proposal.FilterPresetStorage
	hermesMigrator            *migration.HermesMigrator
	servicesManager           *service.Manager
	earningsProvider          earningsProvider
}

type earningsProvider interface {
	GetEarningsDetailed(chainID int64, id identity.Identity) *pingpongEvent.EarningsDetailed
}

type serviceState struct {
	id       service.ID
	isActive bool
}

// MobileNodeOptions contains common mobile node options.
type MobileNodeOptions struct {
	Network                        string
	KeepConnectedOnFail            bool
	DiscoveryAddress               string
	BrokerAddresses                []string
	EtherClientRPCL1               []string
	EtherClientRPCL2               []string
	FeedbackURL                    string
	QualityOracleURL               string
	IPDetectorURL                  string
	LocationDetectorURL            string
	TransactorEndpointAddress      string
	AffiliatorEndpointAddress      string
	HermesEndpointAddress          string
	Chain1ID                       int64
	Chain2ID                       int64
	ActiveChainID                  int64
	PilvytisAddress                string
	MystSCAddress                  string
	RegistrySCAddress              string
	HermesSCAddress                string
	ChannelImplementationSCAddress string
	CacheTTLSeconds                int
	ObserverAddress                string
	IsProvider                     bool
	TequilapiSecured               bool
}

// ConsumerPaymentConfig defines consumer side payment configuration
type ConsumerPaymentConfig struct {
	PriceGiBMax  string
	PriceHourMax string
}

// DefaultNodeOptions returns default options.
func DefaultNodeOptions() *MobileNodeOptions {
	return DefaultNodeOptionsByNetwork(string(config.Mainnet))
}

// DefaultNodeOptionsByNetwork returns default options by network.
func DefaultNodeOptionsByNetwork(network string) *MobileNodeOptions {
	options := &MobileNodeOptions{
		KeepConnectedOnFail: true,
		FeedbackURL:         "https://feedback.mysterium.network",
		QualityOracleURL:    "https://quality.mysterium.network/api/v3",
		IPDetectorURL:       "https://location.mysterium.network/api/v1/location",
		LocationDetectorURL: "https://location.mysterium.network/api/v1/location",
		CacheTTLSeconds:     5,
	}

	bcNetwork, err := config.ParseBlockchainNetwork(network)
	if err != nil {
		bcNetwork = config.Mainnet
	}

	switch bcNetwork {
	case config.Mainnet:
		options.Network = string(config.Mainnet)
		options = setDefinitionOptions(options, metadata.MainnetDefinition)
	case config.Testnet:
		options.Network = string(config.Testnet)
		options = setDefinitionOptions(options, metadata.TestnetDefinition)
	case config.Localnet:
		options.Network = string(config.Localnet)
		options = setDefinitionOptions(options, metadata.LocalnetDefinition)
	}

	return options
}

func setDefinitionOptions(options *MobileNodeOptions, definition metadata.NetworkDefinition) *MobileNodeOptions {
	options.DiscoveryAddress = definition.DiscoveryAddress
	options.BrokerAddresses = definition.BrokerAddresses
	options.EtherClientRPCL1 = definition.Chain1.EtherClientRPC
	options.EtherClientRPCL2 = definition.Chain2.EtherClientRPC
	options.TransactorEndpointAddress = definition.TransactorAddress
	options.AffiliatorEndpointAddress = definition.AffiliatorAddress
	options.ActiveChainID = definition.DefaultChainID
	options.Chain1ID = definition.Chain1.ChainID
	options.Chain2ID = definition.Chain2.ChainID
	options.PilvytisAddress = definition.PilvytisAddress
	options.MystSCAddress = definition.Chain2.MystAddress
	options.RegistrySCAddress = definition.Chain2.RegistryAddress
	options.HermesSCAddress = definition.Chain2.HermesID
	options.ChannelImplementationSCAddress = definition.Chain2.ChannelImplAddress
	options.ObserverAddress = definition.ObserverAddress
	return options
}

// NewNode function creates new Node.
func NewNode(appPath string, options *MobileNodeOptions) (*MobileNode, error) {
	var di cmd.Dependencies

	if appPath == "" {
		return nil, errors.New("node app path is required")
	}

	dataDir := filepath.Join(appPath, ".mysterium")
	config.Current.SetDefault(config.FlagDataDir.Name, dataDir)
	currentDir := appPath
	if err := loadUserConfig(dataDir); err != nil {
		return nil, err
	}

	config.Current.SetDefault(config.FlagChainID.Name, options.ActiveChainID)
	config.Current.SetDefault(config.FlagKeepConnectedOnFail.Name, options.KeepConnectedOnFail)
	config.Current.SetDefault(config.FlagAutoReconnect.Name, "true")
	config.Current.SetDefault(config.FlagDefaultCurrency.Name, metadata.DefaultNetwork.DefaultCurrency)
	config.Current.SetDefault(config.FlagSTUNservers.Name, []string{"stun.l.google.com:19302", "stun1.l.google.com:19302", "stun2.l.google.com:19302"})
	config.Current.SetDefault(config.FlagUDPListenPorts.Name, "10000:60000")
	config.Current.SetDefault(config.FlagStatsReportInterval.Name, time.Second)

	if options.IsProvider {
		config.Current.SetDefault(config.FlagUserspace.Name, "true")
		config.Current.SetDefault(config.FlagAgreedTermsConditions.Name, "true")
		config.Current.SetDefault(config.FlagActiveServices.Name, "wireguard,scraping,data_transfer")
		config.Current.SetDefault(config.FlagDiscoveryPingInterval.Name, "3m")
		config.Current.SetDefault(config.FlagDiscoveryFetchInterval.Name, "3m")
		config.Current.SetDefault(config.FlagAccessPolicyFetchInterval.Name, "10m")
		config.Current.SetDefault(config.FlagPaymentsZeroStakeUnsettledAmount.Name, "5.0")
		config.Current.SetDefault(config.FlagShaperBandwidth.Name, "6250")

		config.Current.SetDefault(config.FlagPaymentsProviderInvoiceFrequency.Name, config.FlagPaymentsProviderInvoiceFrequency.Value)
		config.Current.SetDefault(config.FlagPaymentsLimitProviderInvoiceFrequency.Name, config.FlagPaymentsLimitProviderInvoiceFrequency.Value)
		config.Current.SetDefault(config.FlagPaymentsUnpaidInvoiceValue.Name, config.FlagPaymentsUnpaidInvoiceValue.Value)
		config.Current.SetDefault(config.FlagPaymentsLimitUnpaidInvoiceValue.Name, config.FlagPaymentsLimitUnpaidInvoiceValue.Value)
		config.Current.SetDefault(config.FlagChain1KnownHermeses.Name, config.FlagChain1KnownHermeses.Value)
		config.Current.SetDefault(config.FlagChain2KnownHermeses.Name, config.FlagChain2KnownHermeses.Value)
		config.Current.SetDefault(config.FlagDNSListenPort.Name, config.FlagDNSListenPort.Value)
	}

	bcNetwork, err := config.ParseBlockchainNetwork(options.Network)
	if err != nil {
		return nil, fmt.Errorf("invalid bc network: %w", err)
	}
	config.Current.SetDefaultsByNetwork(bcNetwork)

	network := node.OptionsNetwork{
		Network:          bcNetwork,
		DiscoveryAddress: options.DiscoveryAddress,
		BrokerAddresses:  options.BrokerAddresses,
		EtherClientRPCL1: options.EtherClientRPCL1,
		EtherClientRPCL2: options.EtherClientRPCL2,
		ChainID:          options.ActiveChainID,
		DNSMap: map[string][]string{
			"location.mysterium.network": {"51.158.129.204"},
			"quality.mysterium.network":  {"51.158.129.204"},
			"feedback.mysterium.network": {"116.203.17.150"},
			"api.ipify.org": {
				"54.204.14.42", "54.225.153.147", "54.235.83.248", "54.243.161.145",
				"23.21.109.69", "23.21.126.66",
				"50.19.252.36",
				"174.129.214.20",
			},
		},
	}
	if options.IsProvider {
		network.DNSMap["badupnp.benjojo.co.uk"] = []string{"104.22.70.70", "104.22.71.70", "172.67.25.154"}
	}
	logOptions := logconfig.LogOptions{
		LogLevel: zerolog.DebugLevel,
		LogHTTP:  false,
		Filepath: filepath.Join(dataDir, "mysterium-node"),
	}

	nodeOptions := node.Options{
		LogOptions: logOptions,
		Directories: node.OptionsDirectory{
			Data:     dataDir,
			Storage:  filepath.Join(dataDir, "db"),
			Keystore: filepath.Join(dataDir, "keystore"),
			Runtime:  currentDir,
		},

		SwarmDialerDNSHeadstart: time.Millisecond * 1500,
		Keystore: node.OptionsKeystore{
			UseLightweight: true,
		},
		FeedbackURL:    options.FeedbackURL,
		OptionsNetwork: network,
		Quality: node.OptionsQuality{
			Type:    node.QualityTypeMORQA,
			Address: options.QualityOracleURL,
		},
		Discovery: node.OptionsDiscovery{
			Types:        []node.DiscoveryType{node.DiscoveryTypeAPI},
			Address:      network.DiscoveryAddress,
			FetchEnabled: false,
			DHT: node.OptionsDHT{
				Address:        "0.0.0.0",
				Port:           0,
				Protocol:       "tcp",
				BootstrapPeers: []string{},
			},
		},
		Location: node.OptionsLocation{
			IPDetectorURL: options.IPDetectorURL,
			Type:          node.LocationTypeOracle,
			Address:       options.LocationDetectorURL,
		},
		Transactor: node.OptionsTransactor{
			TransactorEndpointAddress:       options.TransactorEndpointAddress,
			ProviderMaxRegistrationAttempts: 10,
			TransactorFeesValidTime:         time.Second * 30,
		},
		Affiliator: node.OptionsAffiliator{AffiliatorEndpointAddress: options.AffiliatorEndpointAddress},
		Payments: node.OptionsPayments{
			MaxAllowedPaymentPercentile:    1500,
			BCTimeout:                      time.Second * 30,
			SettlementTimeout:              time.Hour * 2,
			HermesStatusRecheckInterval:    time.Hour * 2,
			BalanceFastPollInterval:        time.Second * 30,
			BalanceFastPollTimeout:         time.Minute * 3,
			BalanceLongPollInterval:        time.Hour * 1,
			RegistryTransactorPollInterval: time.Second * 20,
			RegistryTransactorPollTimeout:  time.Minute * 20,
		},
		Chains: node.OptionsChains{
			Chain1: metadata.ChainDefinition{
				RegistryAddress:    options.RegistrySCAddress,
				HermesID:           options.HermesSCAddress,
				ChannelImplAddress: options.ChannelImplementationSCAddress,
				MystAddress:        options.MystSCAddress,
				ChainID:            options.Chain1ID,
			},
			Chain2: metadata.ChainDefinition{
				RegistryAddress:    options.RegistrySCAddress,
				HermesID:           options.HermesSCAddress,
				ChannelImplAddress: options.ChannelImplementationSCAddress,
				MystAddress:        options.MystSCAddress,
				ChainID:            options.Chain2ID,
			},
		},

		Consumer:        true,
		PilvytisAddress: options.PilvytisAddress,
		ObserverAddress: options.ObserverAddress,
		SSE: node.OptionsSSE{
			Enabled: true,
		},
	}
	if options.IsProvider {
		nodeOptions.Discovery.FetchEnabled = true
		nodeOptions.Discovery.PingInterval = config.GetDuration(config.FlagDiscoveryPingInterval)
		nodeOptions.Discovery.FetchInterval = config.GetDuration(config.FlagDiscoveryFetchInterval)
		nodeOptions.Payments = node.OptionsPayments{
			MaxAllowedPaymentPercentile:    3000,
			BCTimeout:                      time.Second * 30,
			SettlementTimeout:              time.Minute * 3,
			SettlementRecheckInterval:      time.Second * 30,
			HermesPromiseSettlingThreshold: 0.01,
			MaxFeeSettlingThreshold:        0.05,
			ConsumerDataLeewayMegabytes:    0x14,
			MinAutoSettleAmount:            5,
			MaxUnSettledAmount:             20,
			HermesStatusRecheckInterval:    time.Hour * 2,
			BalanceFastPollInterval:        time.Second * 30 * 2,
			BalanceFastPollTimeout:         time.Minute * 3 * 3,
			BalanceLongPollInterval:        time.Hour * 1,
			RegistryTransactorPollInterval: time.Second * 20,
			RegistryTransactorPollTimeout:  time.Minute * 20,
			ProviderInvoiceFrequency:       config.GetDuration(config.FlagPaymentsProviderInvoiceFrequency),
			ProviderLimitInvoiceFrequency:  config.GetDuration(config.FlagPaymentsLimitProviderInvoiceFrequency),
			MaxUnpaidInvoiceValue:          config.GetBigInt(config.FlagPaymentsUnpaidInvoiceValue),
			LimitUnpaidInvoiceValue:        config.GetBigInt(config.FlagPaymentsLimitUnpaidInvoiceValue),
		}
		nodeOptions.Payments.LimitUnpaidInvoiceValue = config.GetBigInt(config.FlagPaymentsLimitUnpaidInvoiceValue)
		nodeOptions.Chains.Chain1.KnownHermeses = config.GetStringSlice(config.FlagChain1KnownHermeses)
		nodeOptions.Chains.Chain1.KnownHermeses = config.GetStringSlice(config.FlagChain2KnownHermeses)
		nodeOptions.Consumer = false
		nodeOptions.TequilapiEnabled = true
		nodeOptions.TequilapiPort = 4050
		nodeOptions.TequilapiAddress = "127.0.0.1"
		nodeOptions.TequilapiSecured = options.TequilapiSecured
		nodeOptions.UI = node.OptionsUI{
			UIEnabled:     true,
			UIBindAddress: "127.0.0.1",
			UIPort:        4449,
		}
	}

	err = di.Bootstrap(nodeOptions)
	if err != nil {
		return nil, fmt.Errorf("could not bootstrap dependencies: %w", err)
	}

	mobileNode := &MobileNode{
		shutdown:                  di.Shutdown,
		node:                      di.Node,
		stateKeeper:               di.StateKeeper,
		connectionManager:         di.MultiConnectionManager,
		locationResolver:          di.LocationResolver,
		natProber:                 di.NATProber,
		identitySelector:          di.IdentitySelector,
		signerFactory:             di.SignerFactory,
		ipResolver:                di.IPResolver,
		eventBus:                  di.EventBus,
		connectionRegistry:        di.ConnectionRegistry,
		feedbackReporter:          di.Reporter,
		affiliator:                di.Affiliator,
		transactor:                di.Transactor,
		identityRegistry:          di.IdentityRegistry,
		consumerBalanceTracker:    di.ConsumerBalanceTracker,
		identityChannelCalculator: di.AddressProvider,
		proposalsManager: newProposalsManager(
			di.ProposalRepository,
			di.FilterPresetStorage,
			di.NATProber,
			time.Duration(options.CacheTTLSeconds)*time.Second,
		),
		pilvytis:            di.PilvytisAPI,
		pilvytisOrderIssuer: di.PilvytisOrderIssuer,
		startTime:           time.Now(),
		chainID:             nodeOptions.OptionsNetwork.ChainID,
		sessionStorage:      di.SessionStorage,
		identityMover:       di.IdentityMover,
		entertainmentEstimator: entertainment.NewEstimator(
			config.FlagPaymentPriceGiB.Value,
			config.FlagPaymentPriceHour.Value,
		),
		residentCountry:     di.ResidentCountry,
		filterPresetStorage: di.FilterPresetStorage,
		hermesMigrator:      di.HermesMigrator,
		earningsProvider:    di.HermesChannelRepository,
	}

	if options.IsProvider {
		mobileNode.servicesManager = di.ServicesManager
	}

	return mobileNode, nil
}

// GetDefaultCurrency returns the current default currency set.
func (mb *MobileNode) GetDefaultCurrency() string {
	return config.Current.GetString(config.FlagDefaultCurrency.Name)
}

// ProposalChangeCallback represents proposal callback.
type ProposalChangeCallback interface {
	OnChange(proposal []byte)
}

// GetLocationResponse represents location response.
type GetLocationResponse struct {
	IP      string
	Country string
}

// GetLocation return current location including country and IP.
func (mb *MobileNode) GetLocation() (*GetLocationResponse, error) {
	tr := http.DefaultTransport.(*http.Transport)
	tr.DisableKeepAlives = true
	c := requests.NewHTTPClientWithTransport(tr, 30*time.Second)
	resolver := location.NewOracleResolver(c, DefaultNodeOptions().LocationDetectorURL)
	loc, err := resolver.DetectLocation()
	// TODO this is temporary workaround to show correct location on Android.
	// This needs to be fixed on the di level to make sure we are using correct resolver in transport and in visual part.
	// loc, err := mb.locationResolver.DetectLocation()
	if err != nil {
		return nil, fmt.Errorf("could not get location: %w", err)
	}

	return &GetLocationResponse{
		IP:      loc.IP,
		Country: loc.Country,
	}, nil
}

// SetUserConfig sets a user values in the configuration file
func (mb *MobileNode) SetUserConfig(key, value string) error {
	return setUserConfig(key, value)
}

// GetStatusResponse represents status response.
type GetStatusResponse struct {
	State    string
	Proposal proposalDTO
}

// GetStatus returns current connection state and provider info if connected to VPN.
func (mb *MobileNode) GetStatus() ([]byte, error) {
	status := mb.connectionManager.Status(0)

	resp := &GetStatusResponse{
		State: string(status.State),
		Proposal: proposalDTO{
			ProviderID:   status.Proposal.ProviderID,
			ServiceType:  status.Proposal.ServiceType,
			Country:      status.Proposal.Location.Country,
			IPType:       status.Proposal.Location.IPType,
			QualityLevel: proposalQualityLevel(status.Proposal.Quality.Quality),
			Price: proposalPrice{
				Currency: mb.GetDefaultCurrency(),
			},
		},
	}

	if status.Proposal.Price.PricePerGiB != nil && status.Proposal.Price.PricePerHour != nil {
		resp.Proposal.Price.PerGiB, _ = big.NewFloat(0).SetInt(status.Proposal.Price.PricePerGiB).Float64()
		resp.Proposal.Price.PerHour, _ = big.NewFloat(0).SetInt(status.Proposal.Price.PricePerHour).Float64()
	}

	return json.Marshal(resp)
}

// GetNATType return current NAT type after a probe.
func (mb *MobileNode) GetNATType() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	natType, err := mb.natProber.Probe(ctx)
	return string(natType), err
}

// StatisticsChangeCallback represents statistics callback.
type StatisticsChangeCallback interface {
	OnChange(duration int64, bytesReceived int64, bytesSent int64, tokensSpent float64)
}

// RegisterStatisticsChangeCallback registers callback which is called on active connection
// statistics change.
func (mb *MobileNode) RegisterStatisticsChangeCallback(cb StatisticsChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionStatistics, func(e connectionstate.AppEventConnectionStatistics) {
		var tokensSpent float64
		agreementTotal := mb.stateKeeper.GetConnection("").Invoice.AgreementTotal
		if agreementTotal != nil {
			tokensSpent = units.BigIntWeiToFloatEth(agreementTotal)
		}

		cb.OnChange(int64(e.SessionInfo.Duration().Seconds()), int64(e.Stats.BytesReceived), int64(e.Stats.BytesSent), tokensSpent)
	})
}

// ConnectionStatusChangeCallback represents status callback.
type ConnectionStatusChangeCallback interface {
	OnChange(status string)
}

// ServiceStatusChangeCallback represents status callback.
type ServiceStatusChangeCallback interface {
	OnChange(service string, status string)
}

// RegisterConnectionStatusChangeCallback registers callback which is called on active connection
// status change.
func (mb *MobileNode) RegisterConnectionStatusChangeCallback(cb ConnectionStatusChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, func(e connectionstate.AppEventConnectionState) {
		cb.OnChange(string(e.State))
	})
}

// RegisterServiceStatusChangeCallback registers callback which is called on active connection
// status change.
func (mb *MobileNode) RegisterServiceStatusChangeCallback(cb ServiceStatusChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(servicestate.AppTopicServiceStatus, func(e servicestate.AppEventServiceStatus) {
		cb.OnChange(e.Type, e.Status)
	})
}

// BalanceChangeCallback represents balance change callback.
type BalanceChangeCallback interface {
	OnChange(identityAddress string, balance float64)
}

// RegisterBalanceChangeCallback registers callback which is called on identity balance change.
func (mb *MobileNode) RegisterBalanceChangeCallback(cb BalanceChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(event.AppTopicBalanceChanged, func(e event.AppEventBalanceChanged) {
		balance := crypto.BigMystToFloat(e.Current)
		cb.OnChange(e.Identity.Address, balance)
	})
}

// ConnectRequest represents connect request.
/*
 * DNSOption:
 *	- "auto" (default) tries the following with fallbacks: provider's DNS -> client's system DNS -> public DNS
 *  - "provider" uses DNS servers from provider's system configuration
 *  - "system" uses DNS servers from client's system configuration
 */
type ConnectRequest struct {
	Providers               string // comma separated list of providers that will be used for the connection.
	IdentityAddress         string
	ServiceType             string
	CountryCode             string
	IPType                  string
	SortBy                  string
	DNSOption               string
	IncludeMonitoringFailed bool
}

func (cr *ConnectRequest) dnsOption() (connection.DNSOption, error) {
	if len(cr.DNSOption) > 0 {
		return connection.NewDNSOption(cr.DNSOption)
	}

	return connection.DNSOptionAuto, nil
}

// ConnectResponse represents connect response with optional error code and message.
type ConnectResponse struct {
	ErrorCode    string
	ErrorMessage string
}

const (
	connectErrInvalidProposal     = "InvalidProposal"
	connectErrInsufficientBalance = "InsufficientBalance"
	connectErrUnknown             = "Unknown"
)

// Connect connects to given provider.
func (mb *MobileNode) Connect(req *ConnectRequest) *ConnectResponse {
	var providers []string
	if len(req.Providers) > 0 {
		providers = strings.Split(req.Providers, ",")
	}

	f := &proposal.Filter{
		ServiceType:             req.ServiceType,
		LocationCountry:         req.CountryCode,
		ProviderIDs:             providers,
		IPType:                  req.IPType,
		IncludeMonitoringFailed: req.IncludeMonitoringFailed,
		ExcludeUnsupported:      true,
	}

	proposalLookup := connection.FilteredProposals(f, req.SortBy, mb.proposalsManager.repository)

	qualityEvent := quality.ConnectionEvent{
		ServiceType: req.ServiceType,
		ConsumerID:  req.IdentityAddress,
	}

	if len(req.Providers) == 1 {
		qualityEvent.ProviderID = providers[0]
	}

	dnsOption, err := req.dnsOption()
	if err != nil {
		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}
	connectOptions := connection.ConnectParams{
		DNS: dnsOption,
	}

	hermes, err := mb.identityChannelCalculator.GetActiveHermes(mb.chainID)
	if err != nil {
		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}

	if err := mb.connectionManager.Connect(identity.FromAddress(req.IdentityAddress), hermes, proposalLookup, connectOptions); err != nil {
		qualityEvent.Stage = quality.StageConnectionUnknownError
		qualityEvent.Error = err.Error()
		mb.eventBus.Publish(quality.AppTopicConnectionEvents, qualityEvent)

		if errors.Is(err, connection.ErrInsufficientBalance) {
			return &ConnectResponse{
				ErrorCode: connectErrInsufficientBalance,
			}
		}

		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}

	qualityEvent.Stage = quality.StageConnectionOK
	mb.eventBus.Publish(quality.AppTopicConnectionEvents, qualityEvent)

	return &ConnectResponse{}
}

// Reconnect is deprecated, we are doing reconnect now in the connection manager.
// TODO remove this from mobile app and here too.
func (mb *MobileNode) Reconnect(req *ConnectRequest) *ConnectResponse {
	return &ConnectResponse{}
}

// Disconnect disconnects or cancels current connection.
func (mb *MobileNode) Disconnect() error {
	if err := mb.connectionManager.Disconnect(0); err != nil {
		return fmt.Errorf("could not disconnect: %w", err)
	}

	return nil
}

// GetBalanceRequest represents balance request.
type GetBalanceRequest struct {
	IdentityAddress string
}

// GetBalanceResponse represents balance response.
type GetBalanceResponse struct {
	Balance float64
}

// GetBalance returns current balance.
func (mb *MobileNode) GetBalance(req *GetBalanceRequest) (*GetBalanceResponse, error) {
	balance := mb.consumerBalanceTracker.GetBalance(mb.chainID, identity.FromAddress(req.IdentityAddress))
	b := crypto.BigMystToFloat(balance)

	return &GetBalanceResponse{Balance: b}, nil
}

// GetUnsettledEarnings returns unsettled earnings.
func (mb *MobileNode) GetUnsettledEarnings(req *GetBalanceRequest) (*GetBalanceResponse, error) {
	earnings := mb.earningsProvider.GetEarningsDetailed(mb.chainID, identity.FromAddress(req.IdentityAddress))
	u := crypto.BigMystToFloat(earnings.Total.UnsettledBalance)

	return &GetBalanceResponse{Balance: u}, nil
}

// ForceBalanceUpdate force updates balance and returns the updated balance.
func (mb *MobileNode) ForceBalanceUpdate(req *GetBalanceRequest) *GetBalanceResponse {
	return &GetBalanceResponse{
		Balance: crypto.BigMystToFloat(mb.consumerBalanceTracker.ForceBalanceUpdateCached(mb.chainID, identity.FromAddress(req.IdentityAddress))),
	}
}

// SendFeedbackRequest represents user feedback request.
type SendFeedbackRequest struct {
	Email       string
	Description string
}

// SendFeedback sends user feedback via feedback reported.
func (mb *MobileNode) SendFeedback(req *SendFeedbackRequest) error {
	report := feedback.UserReport{
		Email:       req.Email,
		Description: req.Description,
	}

	result, err := mb.feedbackReporter.NewIssue(report)
	if err != nil {
		return fmt.Errorf("could not create user report: %w", err)
	}

	if !result.Success {
		return errors.New("user report sent but got error response")
	}

	return nil
}

// Shutdown function stops running mobile node.
func (mb *MobileNode) Shutdown() error {
	return mb.shutdown()
}

// WaitUntilDies function returns when node stops.
func (mb *MobileNode) WaitUntilDies() error {
	return mb.node.Wait()
}

// OverrideWireguardConnection overrides default wireguard connection implementation to more mobile adapted one.
func (mb *MobileNode) OverrideWireguardConnection(wgTunnelSetup WireguardTunnelSetup) {
	wireguard.Bootstrap()

	factory := func() (connection.Connection, error) {
		opts := wireGuardOptions{
			statsUpdateInterval: 1 * time.Second,
			handshakeTimeout:    1 * time.Minute,
		}

		return NewWireGuardConnection(
			opts,
			newWireguardDevice(wgTunnelSetup),
			mb.ipResolver,
			wireguard_connection.NewHandshakeWaiter(),
		)
	}
	mb.connectionRegistry.Register(wireguard.ServiceType, factory)

	router.SetProtectFunc(wgTunnelSetup.Protect)
}

// HealthCheckData represents node health check info.
type HealthCheckData struct {
	Uptime    string     `json:"uptime"`
	Version   string     `json:"version"`
	BuildInfo *BuildInfo `json:"build_info"`
}

// BuildInfo represents node build info.
type BuildInfo struct {
	Commit      string `json:"commit"`
	Branch      string `json:"branch"`
	BuildNumber string `json:"build_number"`
}

// HealthCheck returns node health check data.
func (mb *MobileNode) HealthCheck() *HealthCheckData {
	return &HealthCheckData{
		Uptime:  time.Since(mb.startTime).String(),
		Version: metadata.VersionAsString(),
		BuildInfo: &BuildInfo{
			Commit:      metadata.BuildCommit,
			Branch:      metadata.BuildBranch,
			BuildNumber: metadata.BuildNumber,
		},
	}
}
