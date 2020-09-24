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
	"math/big"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/core/connection/connectionstate"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/core/port"
	"github.com/mysteriumnetwork/node/core/state"
	"github.com/mysteriumnetwork/node/identity/registry"
	wireguard_connection "github.com/mysteriumnetwork/node/services/wireguard/connection"
	"github.com/mysteriumnetwork/node/session/pingpong"
	"github.com/mysteriumnetwork/node/session/pingpong/event"
	"github.com/mysteriumnetwork/payments/crypto"

	"github.com/rs/zerolog"

	"github.com/mysteriumnetwork/node/cmd"
	"github.com/mysteriumnetwork/node/core/connection"
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
	"github.com/mysteriumnetwork/node/services/wireguard"
)

// MobileNode represents node object tuned for mobile devices
type MobileNode struct {
	shutdown                     func() error
	node                         *cmd.Node
	stateKeeper                  *state.Keeper
	connectionManager            connection.Manager
	locationResolver             *location.Cache
	identitySelector             selector.Handler
	signerFactory                identity.SignerFactory
	ipResolver                   ip.Resolver
	eventBus                     eventbus.EventBus
	connectionRegistry           *connection.Registry
	proposalsManager             *proposalsManager
	hermes                       common.Address
	feedbackReporter             *feedback.Reporter
	transactor                   *registry.Transactor
	identityRegistry             registry.IdentityRegistry
	identityChannelCalculator    *pingpong.ChannelAddressCalculator
	consumerBalanceTracker       *pingpong.ConsumerBalanceTracker
	registryAddress              string
	channelImplementationAddress string
	startTime                    time.Time
}

// MobileNodeOptions contains common mobile node options.
type MobileNodeOptions struct {
	Betanet                         bool
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
	HermesEndpointAddress           string
	HermesID                        string
	MystSCAddress                   string
}

// DefaultNodeOptions returns default options.
func DefaultNodeOptions() *MobileNodeOptions {
	return &MobileNodeOptions{
		Betanet:                         true,
		ExperimentNATPunching:           true,
		MysteriumAPIAddress:             metadata.BetanetDefinition.MysteriumAPIAddress,
		BrokerAddress:                   metadata.BetanetDefinition.BrokerAddress,
		EtherClientRPC:                  metadata.BetanetDefinition.EtherClientRPC,
		FeedbackURL:                     "https://feedback.mysterium.network",
		QualityOracleURL:                "https://betanet-quality.mysterium.network/api/v1",
		IPDetectorURL:                   "https://api.ipify.org/?format=json",
		LocationDetectorURL:             "https://betanet-location.mysterium.network/api/v1/location",
		TransactorEndpointAddress:       metadata.BetanetDefinition.TransactorAddress,
		TransactorRegistryAddress:       metadata.BetanetDefinition.RegistryAddress,
		TransactorChannelImplementation: metadata.BetanetDefinition.ChannelImplAddress,
		HermesID:                        metadata.BetanetDefinition.HermesID,
		MystSCAddress:                   "0xf74a5ca65E4552CfF0f13b116113cCb493c580C5",
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
		Betanet:               options.Betanet,
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

		Keystore: node.OptionsKeystore{
			UseLightweight: true,
		},
		UI: node.OptionsUI{
			UIEnabled: false,
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
			ProviderRegistrationStake:       big.NewInt(6200000000),
		},
		Hermes: node.OptionsHermes{
			HermesID: options.HermesID,
		},
		Payments: node.OptionsPayments{
			MaxAllowedPaymentPercentile:    1500,
			BCTimeout:                      time.Second * 30,
			HermesPromiseSettlingThreshold: 0.1,
			SettlementTimeout:              time.Hour * 2,
			MystSCAddress:                  options.MystSCAddress,
		},
		Consumer: true,
		P2PPorts: port.UnspecifiedRange(),
	}

	err := di.Bootstrap(nodeOptions)
	if err != nil {
		return nil, errors.Wrap(err, "could not bootstrap dependencies")
	}

	mobileNode := &MobileNode{
		shutdown:                     func() error { return di.Shutdown() },
		node:                         di.Node,
		stateKeeper:                  di.StateKeeper,
		connectionManager:            di.ConnectionManager,
		locationResolver:             di.LocationResolver,
		identitySelector:             di.IdentitySelector,
		signerFactory:                di.SignerFactory,
		ipResolver:                   di.IPResolver,
		eventBus:                     di.EventBus,
		connectionRegistry:           di.ConnectionRegistry,
		hermes:                       common.HexToAddress(nodeOptions.Hermes.HermesID),
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
		),
		startTime: time.Now(),
	}
	return mobileNode, nil
}

// GetProposals returns service proposals from API or cache. Proposals returned as JSON byte array since
// go mobile does not support complex slices.
func (mb *MobileNode) GetProposals(req *GetProposalsRequest) ([]byte, error) {
	return mb.proposalsManager.getProposals(req)
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
	OnChange(duration int64, bytesReceived int64, bytesSent int64, tokensSpent float64)
}

// RegisterStatisticsChangeCallback registers callback which is called on active connection
// statistics change.
func (mb *MobileNode) RegisterStatisticsChangeCallback(cb StatisticsChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionStatistics, func(e connectionstate.AppEventConnectionStatistics) {
		tokensSpent := crypto.BigMystToFloat(mb.stateKeeper.GetState().Connection.Invoice.AgreementTotal)
		cb.OnChange(int64(e.SessionInfo.Duration().Seconds()), int64(e.Stats.BytesReceived), int64(e.Stats.BytesSent), tokensSpent)
	})
}

// ConnectionStatusChangeCallback represents status callback.
type ConnectionStatusChangeCallback interface {
	OnChange(status string)
}

// RegisterConnectionStatusChangeCallback registers callback which is called on active connection
// status change.
func (mb *MobileNode) RegisterConnectionStatusChangeCallback(cb ConnectionStatusChangeCallback) {
	_ = mb.eventBus.SubscribeAsync(connectionstate.AppTopicConnectionState, func(e connectionstate.AppEventConnectionState) {
		cb.OnChange(string(e.State))
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
	ForceReconnect    bool
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
	if err := mb.connectionManager.Connect(identity.FromAddress(req.IdentityAddress), mb.hermes, *proposal, connectOptions); err != nil {
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

// Reconnect checks weather session is alive and reconnects if its dead. Force reconnect if ForceReconnect is set.
func (mb *MobileNode) Reconnect(req *ConnectRequest) *ConnectResponse {
	reconnect := func() *ConnectResponse {
		if err := mb.Disconnect(); err != nil {
			log.Err(err).Msg("Failed to disconnect previous session")
		}
		return mb.Connect(req)
	}
	if req.ForceReconnect {
		log.Info().Msg("Forcing immediate reconnect")
		return reconnect()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := mb.connectionManager.CheckChannel(ctx); err != nil {
		log.Info().Msgf("Forcing reconnect after failed channel: %s", err)
		return reconnect()
	}

	log.Info().Msg("Reconnect is not needed - p2p channel is alive")
	return &ConnectResponse{}
}

// Disconnect disconnects or cancels current connection.
func (mb *MobileNode) Disconnect() error {
	if err := mb.connectionManager.Disconnect(); err != nil {
		return errors.Wrap(err, "could not disconnect")
	}
	return nil
}

// GetIdentityRequest represents identity request.
type GetIdentityRequest struct {
	Address    string
	Passphrase string
}

// GetIdentityResponse represents identity response.
type GetIdentityResponse struct {
	IdentityAddress    string
	ChannelAddress     string
	RegistrationStatus string
}

// GetIdentity finds first identity and unlocks it.
// If there is no identity default one will be created.
func (mb *MobileNode) GetIdentity(req *GetIdentityRequest) (*GetIdentityResponse, error) {
	if req == nil {
		req = &GetIdentityRequest{}
	}
	id, err := mb.identitySelector.UseOrCreate(req.Address, req.Passphrase)
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
	Fee float64
}

// GetIdentityRegistrationFees returns identity registration fees.
func (mb *MobileNode) GetIdentityRegistrationFees() (*GetIdentityRegistrationFeesResponse, error) {
	fees, err := mb.transactor.FetchRegistrationFees()
	if err != nil {
		return nil, errors.Wrap(err, "could not get registration fees")
	}

	fee := crypto.BigMystToFloat(fees.Fee)

	return &GetIdentityRegistrationFeesResponse{Fee: fee}, nil
}

// RegisterIdentityRequest represents identity registration request.
type RegisterIdentityRequest struct {
	IdentityAddress string
}

// RegisterIdentity starts identity registration in background.
func (mb *MobileNode) RegisterIdentity(req *RegisterIdentityRequest) error {
	fees, err := mb.transactor.FetchRegistrationFees()
	if err != nil {
		return errors.Wrap(err, "could not get registration fees")
	}

	err = mb.transactor.RegisterIdentity(req.IdentityAddress, big.NewInt(0), fees.Fee, "")
	if err != nil {
		return errors.Wrap(err, "could not register identity")
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
	balance := mb.consumerBalanceTracker.GetBalance(identity.FromAddress(req.IdentityAddress))
	b := crypto.BigMystToFloat(balance)
	return &GetBalanceResponse{Balance: b}, nil
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
			wireguard_connection.NewHandshakeWaiter(),
		)
	}
	mb.connectionRegistry.Register(wireguard.ServiceType, factory)
}

// HealthCheckData represents node health check info
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
