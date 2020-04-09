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
	"fmt"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/identity/registry"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/connection"
	"github.com/mysteriumnetwork/node/core/discovery"
	"github.com/mysteriumnetwork/node/core/discovery/proposal"
	"github.com/mysteriumnetwork/node/core/ip"
	"github.com/mysteriumnetwork/node/core/location"
	"github.com/mysteriumnetwork/node/core/node"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/feedback"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/identity/selector"
	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/mysteriumnetwork/node/market"
	"github.com/mysteriumnetwork/node/metadata"
	"github.com/mysteriumnetwork/node/services/openvpn"
	"github.com/mysteriumnetwork/node/services/wireguard"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// MobileNode represents node object tuned for mobile devices
type MobileNode struct {
	shutdown                     func() error
	node                         *node.Node
	connectionManager            connection.Manager
	locationResolver             *location.Cache
	identitySelector             selector.Handler
	signerFactory                identity.SignerFactory
	natPinger                    natPinger
	ipResolver                   ip.Resolver
	eventBus                     eventbus.EventBus
	connectionRegistry           *connection.Registry
	proposalsManager             *proposalsManager
	accountant                   common.Address
	feedbackReporter             *feedback.Reporter
	transactor                   *registry.Transactor
	identityRegistry             registry.IdentityRegistry
	identityChannelCalculator    *pingpong.ChannelAddressCalculator
	consumerBalanceTracker       *pingpong.ConsumerBalanceTracker
	registryAddress              string
	channelImplementationAddress string
}

// MobileNodeOptions contains common mobile node options.
type MobileNodeOptions struct {
	Testnet                         bool
	Localnet                        bool
	ExperimentNATPunching           bool
	MysteriumAPIAddress             string
	BrokerAddress                   string
	EtherClientRPC                  string
	FeedbackURL                     string
	QualityOracleURL                string
	IPDetectorURL                   string
	LocationDetectorURL             string
	TransactorEndpointAddress       string
	TransactorRegistryAddress       string
	TransactorChannelImplementation string
	AccountantEndpointAddress       string
	AccountantID                    string
	MystSCAddress                   string
}

// DefaultNodeOptions returns default options.
func DefaultNodeOptions() *MobileNodeOptions {
	return &MobileNodeOptions{
		Testnet:                         true,
		ExperimentNATPunching:           true,
		MysteriumAPIAddress:             metadata.TestnetDefinition.MysteriumAPIAddress,
		BrokerAddress:                   metadata.TestnetDefinition.BrokerAddress,
		EtherClientRPC:                  metadata.TestnetDefinition.EtherClientRPC,
		FeedbackURL:                     "https://feedback.mysterium.network",
		QualityOracleURL:                "https://quality.mysterium.network/api/v1",
		IPDetectorURL:                   "https://api.ipify.org/?format=json",
		LocationDetectorURL:             "https://testnet-location.mysterium.network/api/v1/location",
		TransactorEndpointAddress:       metadata.TestnetDefinition.TransactorAddress,
		TransactorRegistryAddress:       metadata.TestnetDefinition.RegistryAddress,
		TransactorChannelImplementation: metadata.TestnetDefinition.ChannelImplAddress,
		AccountantEndpointAddress:       metadata.TestnetDefinition.AccountantAddress,
		AccountantID:                    metadata.TestnetDefinition.AccountantID,
		MystSCAddress:                   "0x7753cfAD258eFbC52A9A1452e42fFbce9bE486cb",
	}
}

// NewNode function creates new Node
func NewNode(appPath string, options *MobileNodeOptions) (*MobileNode, error) {
	var di cmd.Dependencies

	if appPath == "" {
		return nil, errors.New("node app path is required")
	}
	dataDir := filepath.Join(appPath, ".mysterium")
	currentDir := appPath

	network := node.OptionsNetwork{
		Testnet:               options.Testnet,
		Localnet:              options.Localnet,
		ExperimentNATPunching: options.ExperimentNATPunching,
		MysteriumAPIAddress:   options.MysteriumAPIAddress,
		BrokerAddress:         options.BrokerAddress,
		EtherClientRPC:        options.EtherClientRPC,
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

		TequilapiEnabled: false,

		Openvpn: embeddedLibCheck{},

		Keystore: node.OptionsKeystore{
			UseLightweight: true,
		},
		UI: node.OptionsUI{
			UIEnabled: false,
		},
		MMN: node.OptionsMMN{
			Enabled: false,
		},
		FeedbackURL:    options.FeedbackURL,
		OptionsNetwork: network,
		Quality: node.OptionsQuality{
			Type:    node.QualityTypeMORQA,
			Address: options.QualityOracleURL,
		},
		Discovery: node.OptionsDiscovery{
			Types:        []node.DiscoveryType{node.DiscoveryTypeAPI, node.DiscoveryTypeBroker},
			Address:      network.MysteriumAPIAddress,
			FetchEnabled: false,
		},
		Location: node.OptionsLocation{
			IPDetectorURL: options.IPDetectorURL,
			Type:          node.LocationTypeOracle,
			Address:       options.LocationDetectorURL,
		},
		Transactor: node.OptionsTransactor{
			TransactorEndpointAddress:       options.TransactorEndpointAddress,
			RegistryAddress:                 options.TransactorRegistryAddress,
			ChannelImplementation:           options.TransactorChannelImplementation,
			ProviderMaxRegistrationAttempts: 10,
			ProviderRegistrationRetryDelay:  time.Minute * 3,
			ProviderRegistrationStake:       6200000000,
		},
		Accountant: node.OptionsAccountant{
			AccountantEndpointAddress: options.AccountantEndpointAddress,
			AccountantID:              options.AccountantID,
		},
		Payments: node.OptionsPayments{
			MaxAllowedPaymentPercentile:        1500,
			BCTimeout:                          time.Second * 30,
			AccountantPromiseSettlingThreshold: 0.1,
			SettlementTimeout:                  time.Hour * 2,
			MystSCAddress:                      options.MystSCAddress,
			ConsumerLowerMinutePriceBound:      0,
			ConsumerUpperMinutePriceBound:      50000,
			ConsumerLowerGBPriceBound:          0,
			ConsumerUpperGBPriceBound:          7000000,
		},
		MobileConsumer: true,
	}

	err := di.Bootstrap(nodeOptions)
	if err != nil {
		return nil, errors.Wrap(err, "could not bootstrap dependencies")
	}

	mobileNode := &MobileNode{
		shutdown:                     func() error { return di.Shutdown() },
		node:                         di.Node,
		connectionManager:            di.ConnectionManager,
		locationResolver:             di.LocationResolver,
		identitySelector:             di.IdentitySelector,
		signerFactory:                di.SignerFactory,
		natPinger:                    di.NATPinger,
		ipResolver:                   di.IPResolver,
		eventBus:                     di.EventBus,
		connectionRegistry:           di.ConnectionRegistry,
		accountant:                   common.HexToAddress(nodeOptions.Accountant.AccountantID),
		feedbackReporter:             di.Reporter,
		transactor:                   di.Transactor,
		identityRegistry:             di.IdentityRegistry,
		consumerBalanceTracker:       di.ConsumerBalanceTracker,
		identityChannelCalculator:    di.ChannelAddressCalculator,
		channelImplementationAddress: nodeOptions.Transactor.ChannelImplementation,
		registryAddress:              nodeOptions.Transactor.RegistryAddress,
		proposalsManager: newProposalsManager(
			di.ProposalRepository,
			di.MysteriumAPI,
			di.QualityClient,
			&proposal.Filter{
				UpperTimePriceBound: &nodeOptions.Payments.ConsumerUpperMinutePriceBound,
				LowerTimePriceBound: &nodeOptions.Payments.ConsumerLowerMinutePriceBound,
				UpperGBPriceBound:   &nodeOptions.Payments.ConsumerUpperGBPriceBound,
				LowerGBPriceBound:   &nodeOptions.Payments.ConsumerLowerGBPriceBound,
				ExcludeUnsupported:  true,
			},
		),
	}
	return mobileNode, nil
}

// GetProposals returns service proposals from API or cache. Proposals returned as JSON byte array since
// go mobile does not support complex slices.
func (mb *MobileNode) GetProposals(req *GetProposalsRequest) ([]byte, error) {
	return mb.proposalsManager.getProposals(req)
}

// GetProposal returns service proposal from cache.
func (mb *MobileNode) GetProposal(req *GetProposalRequest) ([]byte, error) {
	status := mb.connectionManager.Status()
	proposal, err := mb.proposalsManager.getProposal(req)
	if err != nil {
		return nil, errors.Wrap(err, "could not get proposal")
	}
	if proposal == nil {
		return nil, fmt.Errorf("proposal %s-%s not found", status.Proposal.ProviderID, status.Proposal.ServiceType)
	}
	return proposal, nil
}

// ProposalChangeCallback represents proposal callback.
type ProposalChangeCallback interface {
	OnChange(proposal []byte)
}

// RegisterProposalAddedCallback registers callback which is called on newly announced proposals
func (mb *MobileNode) RegisterProposalAddedCallback(cb ProposalChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(discovery.AppTopicProposalAdded, func(proposal market.ServiceProposal) {
		proposalPayload, err := mb.proposalsManager.mapToProposalResponse(&proposal)
		log.Error().Err(err).Msg("Proposal mapping failed")
		cb.OnChange(proposalPayload)
	})
}

// RegisterProposalUpdatedCallback registers callback which is called on re-announced proposals
func (mb *MobileNode) RegisterProposalUpdatedCallback(cb ProposalChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(discovery.AppTopicProposalUpdated, func(proposal market.ServiceProposal) {
		proposalPayload, err := mb.proposalsManager.mapToProposalResponse(&proposal)
		log.Error().Err(err).Msg("Proposal mapping failed")
		cb.OnChange(proposalPayload)
	})
}

// RegisterProposalRemovedCallback registers callback which is called on de-announced proposals
func (mb *MobileNode) RegisterProposalRemovedCallback(cb ProposalChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(discovery.AppTopicProposalRemoved, func(proposal market.ServiceProposal) {
		proposalPayload, err := mb.proposalsManager.mapToProposalResponse(&proposal)
		log.Error().Err(err).Msg("Proposal mapping failed")
		cb.OnChange(proposalPayload)
	})
}

// GetLocationResponse represents location response.
type GetLocationResponse struct {
	IP      string
	Country string
}

// GetLocation return current location including country and IP.
func (mb *MobileNode) GetLocation() (*GetLocationResponse, error) {
	loc, err := mb.locationResolver.DetectLocation()
	if err != nil {
		return nil, errors.Wrap(err, "could not get location")
	}

	return &GetLocationResponse{
		IP:      loc.IP,
		Country: loc.Country,
	}, nil
}

// GetStatusResponse represents status response.
type GetStatusResponse struct {
	State       string
	ProviderID  string
	ServiceType string
}

// GetStatus returns current connection state and provider info if connected to VPN.
func (mb *MobileNode) GetStatus() *GetStatusResponse {
	status := mb.connectionManager.Status()
	return &GetStatusResponse{
		State:       string(status.State),
		ProviderID:  status.Proposal.ProviderID,
		ServiceType: status.Proposal.ServiceType,
	}
}

// StatisticsChangeCallback represents statistics callback.
type StatisticsChangeCallback interface {
	OnChange(duration int64, bytesReceived int64, bytesSent int64)
}

// RegisterStatisticsChangeCallback registers callback which is called on active connection
// statistics change.
func (mb *MobileNode) RegisterStatisticsChangeCallback(cb StatisticsChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connection.AppTopicConnectionStatistics, func(e connection.AppEventConnectionStatistics) {
		cb.OnChange(int64(e.SessionInfo.Duration().Seconds()), int64(e.Stats.BytesReceived), int64(e.Stats.BytesSent))
	})
}

// ConnectionStatusChangeCallback represents status callback.
type ConnectionStatusChangeCallback interface {
	OnChange(status string)
}

// RegisterConnectionStatusChangeCallback registers callback which is called on active connection
// status change.
func (mb *MobileNode) RegisterConnectionStatusChangeCallback(cb ConnectionStatusChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connection.AppTopicConnectionState, func(e connection.AppEventConnectionState) {
		cb.OnChange(string(e.State))
	})
}

// BalanceChangeCallback represents balance change callback.
type BalanceChangeCallback interface {
	OnChange(identityAddress string, balance int64)
}

// RegisterBalanceChangeCallback registers callback which is called on identity balance change.
func (mb *MobileNode) RegisterBalanceChangeCallback(cb BalanceChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(event.AppTopicBalanceChanged, func(e event.AppEventBalanceChanged) {
		cb.OnChange(e.Identity.Address, int64(e.Current))
	})
}

// IdentityRegistrationChangeCallback represents identity registration status callback.
type IdentityRegistrationChangeCallback interface {
	OnChange(identityAddress string, status string)
}

// RegisterIdentityRegistrationChangeCallback registers callback which is called on identity registration status change.
func (mb *MobileNode) RegisterIdentityRegistrationChangeCallback(cb IdentityRegistrationChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(registry.AppTopicIdentityRegistration, func(e registry.AppEventIdentityRegistration) {
		cb.OnChange(e.ID.Address, e.Status.String())
	})
}

// ConnectRequest represents connect request.
type ConnectRequest struct {
	IdentityAddress   string
	ProviderID        string
	ServiceType       string
	DisableKillSwitch bool
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
	proposal, err := mb.proposalsManager.repository.Proposal(market.ProposalID{
		ProviderID:  req.ProviderID,
		ServiceType: req.ServiceType,
	})
	if err != nil {
		return &ConnectResponse{
			ErrorCode:    connectErrInvalidProposal,
			ErrorMessage: err.Error(),
		}
	}

	connectOptions := connection.ConnectParams{
		DisableKillSwitch: req.DisableKillSwitch,
		DNS:               connection.DNSOptionAuto,
	}
	if err := mb.connectionManager.Connect(identity.FromAddress(req.IdentityAddress), mb.accountant, *proposal, connectOptions); err != nil {
		if err == connection.ErrInsufficientBalance {
			return &ConnectResponse{
				ErrorCode: connectErrInsufficientBalance,
			}
		}
		return &ConnectResponse{
			ErrorCode:    connectErrUnknown,
			ErrorMessage: err.Error(),
		}
	}
	return &ConnectResponse{}
}

// Disconnect disconnects or cancels current connection.
func (mb *MobileNode) Disconnect() error {
	if err := mb.connectionManager.Disconnect(); err != nil {
		return errors.Wrap(err, "could not disconnect")
	}
	return nil
}

// GetIdentityResponse represents identity response.
type GetIdentityResponse struct {
	IdentityAddress    string
	ChannelAddress     string
	RegistrationStatus string
}

// GetIdentity finds first identity and unlocks it.
// If there is no identity default one will be created.
func (mb *MobileNode) GetIdentity() (*GetIdentityResponse, error) {
	id, err := mb.identitySelector.UseOrCreate("", "")
	if err != nil {
		return nil, errors.Wrap(err, "could not unlock identity")
	}

	channelAddress, err := mb.identityChannelCalculator.GetChannelAddress(id)
	if err != nil {
		return nil, errors.Wrap(err, "could not generate channel address")
	}

	status, err := mb.identityRegistry.GetRegistrationStatus(id)
	if err != nil {
		return nil, errors.Wrap(err, "could not get identity registration status")
	}

	return &GetIdentityResponse{
		IdentityAddress:    id.Address,
		ChannelAddress:     channelAddress.Hex(),
		RegistrationStatus: status.String(),
	}, nil
}

// GetIdentityRegistrationFeesResponse represents identity registration fees result.
type GetIdentityRegistrationFeesResponse struct {
	Fee int64
}

// GetIdentityRegistrationFees returns identity registration fees.
func (mb *MobileNode) GetIdentityRegistrationFees() (*GetIdentityRegistrationFeesResponse, error) {
	fees, err := mb.transactor.FetchRegistrationFees()
	if err != nil {
		return nil, errors.Wrap(err, "could not get registration fees")
	}
	return &GetIdentityRegistrationFeesResponse{Fee: int64(fees.Fee)}, nil
}

// RegisterIdentityRequest represents identity registration request.
type RegisterIdentityRequest struct {
	IdentityAddress string
	Fee             int64
}

// RegisterIdentity starts identity registration in background.
func (mb *MobileNode) RegisterIdentity(req *RegisterIdentityRequest) error {
	err := mb.transactor.RegisterIdentity(req.IdentityAddress, &registry.IdentityRegistrationRequestDTO{
		Stake:       0,
		Beneficiary: "",
		Fee:         uint64(req.Fee),
	})
	if err != nil {
		return errors.Wrap(err, "could not register identity")
	}
	return nil
}

// TopUpRequest represents top-up request.
type TopUpRequest struct {
	IdentityAddress string
}

// TopUp adds resets to default balance. This is temporary flow while
// payments are not production ready.
func (mb *MobileNode) TopUp(req *TopUpRequest) error {
	if err := mb.transactor.TopUp(req.IdentityAddress); err != nil {
		return errors.Wrap(err, "could not top-up balance")
	}
	return nil
}

// GetBalanceRequest represents balance request.
type GetBalanceRequest struct {
	IdentityAddress string
}

// GetBalanceResponse represents balance response.
type GetBalanceResponse struct {
	Balance int64
}

// GetBalance returns current balance.
func (mb *MobileNode) GetBalance(req *GetBalanceRequest) (*GetBalanceResponse, error) {
	balance := mb.consumerBalanceTracker.GetBalance(identity.FromAddress(req.IdentityAddress))
	return &GetBalanceResponse{Balance: int64(balance)}, nil
}

// SendFeedbackRequest represents user feedback request.
type SendFeedbackRequest struct {
	Description string
}

// SendFeedback sends user feedback via feedback reported.
func (mb *MobileNode) SendFeedback(req *SendFeedbackRequest) error {
	report := feedback.UserReport{
		Description: req.Description,
	}
	result, err := mb.feedbackReporter.NewIssue(report)
	if err != nil {
		return errors.Wrap(err, "could not create user report")
	}

	if !result.Success {
		return errors.New("user report sent but got error response")
	}
	return nil
}

// Shutdown function stops running mobile node
func (mb *MobileNode) Shutdown() error {
	return mb.shutdown()
}

// WaitUntilDies function returns when node stops
func (mb *MobileNode) WaitUntilDies() error {
	return mb.node.Wait()
}

// OverrideOpenvpnConnection replaces default openvpn connection factory with mobile related one returning session that can be reconnected
func (mb *MobileNode) OverrideOpenvpnConnection(tunnelSetup Openvpn3TunnelSetup) ReconnectableSession {
	openvpn.Bootstrap()

	st := &sessionTracker{}
	factory := func() (connection.Connection, error) {
		return NewOpenVPNConnection(
			st,
			mb.signerFactory,
			tunnelSetup,
			mb.natPinger,
			mb.ipResolver,
		)
	}
	_ = mb.eventBus.Subscribe(connection.AppTopicConnectionState, st.handleState)
	mb.connectionRegistry.Register("openvpn", factory)
	return st
}

// OverrideWireguardConnection overrides default wireguard connection implementation to more mobile adapted one
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
			mb.natPinger,
			wireguard_connection.NewHandshakeWaiter(),
		)
	}
	mb.connectionRegistry.Register(wireguard.ServiceType, factory)
}
