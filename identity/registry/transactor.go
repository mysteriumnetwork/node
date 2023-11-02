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

package registry

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/payments/client"
	pc "github.com/mysteriumnetwork/payments/crypto"
	"github.com/mysteriumnetwork/payments/registration"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

// AppTopicTransactorRegistration represents the registration topic to which events regarding registration attempts on transactor will occur
const AppTopicTransactorRegistration = "transactor_identity_registration"

type channelProvider interface {
	GetProviderChannel(chainID int64, hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
	GetLastRegistryNonce(chainID int64, registry common.Address) (*big.Int, error)
}

// AddressProvider provides sc addresses.
type AddressProvider interface {
	GetActiveChannelImplementation(chainID int64) (common.Address, error)
	GetActiveHermes(chainID int64) (common.Address, error)
	GetRegistryAddress(chainID int64) (common.Address, error)
	GetKnownHermeses(chainID int64) ([]common.Address, error)
	GetChannelImplementationForHermes(chainID int64, hermes common.Address) (common.Address, error)
	GetMystAddress(chainID int64) (common.Address, error)
}

type feeType uint8

const (
	settleFeeType        feeType = iota
	registrationFeeType          = 1
	stakeDecreaseFeeType         = 2
)

type feeCacher struct {
	validityDuration time.Duration
	feesMap          map[int64]map[feeType]feeCache
}

type feeCache struct {
	FeesResponse
	CacheValidUntil time.Time
}

func newFeeCacher(validityDuration time.Duration) *feeCacher {
	return &feeCacher{
		validityDuration: validityDuration,
		feesMap:          make(map[int64]map[feeType]feeCache),
	}
}

func (f *feeCacher) getCachedFee(chainId int64, feeType feeType) *FeesResponse {
	if chainFees, ok := f.feesMap[chainId]; ok {
		if fees, ok := chainFees[feeType]; ok {
			if fees.CacheValidUntil.After(time.Now()) {
				return &fees.FeesResponse
			}
		}
	}
	return nil
}

func (f *feeCacher) cacheFee(chainId int64, ftype feeType, response FeesResponse) {
	_, ok := f.feesMap[chainId]
	if !ok {
		f.feesMap[chainId] = make(map[feeType]feeCache)
	}
	cacheExpiration := time.Now().Add(f.validityDuration)
	feeCache := feeCache{
		FeesResponse:    response,
		CacheValidUntil: cacheExpiration,
	}
	if feeCache.CacheValidUntil.After(response.ValidUntil) {
		feeCache.CacheValidUntil = response.ValidUntil
	}
	f.feesMap[chainId][ftype] = feeCache
}

// Transactor allows for convenient calls to the transactor service
type Transactor struct {
	httpClient      *requests.HTTPClient
	endpointAddress string
	signerFactory   identity.SignerFactory
	publisher       eventbus.Publisher
	bc              channelProvider
	addresser       AddressProvider
	feeCache        *feeCacher
}

// NewTransactor creates and returns new Transactor instance
func NewTransactor(httpClient *requests.HTTPClient, endpointAddress string, addresser AddressProvider, signerFactory identity.SignerFactory, publisher eventbus.Publisher, bc channelProvider, feesValidTime time.Duration) *Transactor {
	return &Transactor{
		httpClient:      httpClient,
		endpointAddress: endpointAddress,
		signerFactory:   signerFactory,
		addresser:       addresser,
		publisher:       publisher,
		bc:              bc,
		feeCache:        newFeeCacher(feesValidTime),
	}
}

// FeesResponse represents fees applied by Transactor
type FeesResponse struct {
	Fee        *big.Int  `json:"fee"`
	ValidUntil time.Time `json:"valid_until"`
}

// IsValid returns false if the fee has already expired and should be re-requested
func (fr FeesResponse) IsValid() bool {
	return time.Now().UTC().Before(fr.ValidUntil.UTC())
}

// IdentityRegistrationRequest represents the identity registration request body
type IdentityRegistrationRequest struct {
	RegistryAddress string `json:"registryAddress"`
	HermesID        string `json:"hermesID"`
	// Stake is used by Provider, default 0
	Stake *big.Int `json:"stake"`
	// Fee: negotiated fee with transactor
	Fee *big.Int `json:"fee"`
	// Beneficiary: Provider channelID by default, optionally set during Identity registration.
	// Can be updated later through transactor. We can check it's value directly from SC.
	// Its a cache out address
	Beneficiary string `json:"beneficiary"`
	// Signature from fields above
	Signature string `json:"signature"`
	Identity  string `json:"identity"`
	ChainID   int64  `json:"chainID"`
}

// PromiseSettlementRequest represents the settlement request body
type PromiseSettlementRequest struct {
	HermesID      string   `json:"hermesID"`
	ChannelID     string   `json:"channelID"`
	Amount        *big.Int `json:"amount"`
	TransactorFee *big.Int `json:"fee"`
	Preimage      string   `json:"preimage"`
	Signature     string   `json:"signature"`
	ProviderID    string   `json:"providerID"`
	ChainID       int64    `json:"chainID"`
}

// FetchRegistrationFees fetches current transactor registration fees
func (t *Transactor) FetchRegistrationFees(chainID int64) (FeesResponse, error) {
	cachedFees := t.feeCache.getCachedFee(chainID, registrationFeeType)
	if cachedFees != nil {
		return *cachedFees, nil
	}

	f := FeesResponse{}
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("fee/%v/register", chainID), nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	if err == nil {
		t.feeCache.cacheFee(chainID, registrationFeeType, f)
	}
	return f, err
}

// FetchSettleFees fetches current transactor settlement fees
func (t *Transactor) FetchSettleFees(chainID int64) (FeesResponse, error) {
	cachedFees := t.feeCache.getCachedFee(chainID, settleFeeType)
	if cachedFees != nil {
		return *cachedFees, nil
	}

	f := FeesResponse{}
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("fee/%v/settle", chainID), nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	if err == nil {
		t.feeCache.cacheFee(chainID, settleFeeType, f)
	}
	return f, err
}

// FetchStakeDecreaseFee fetches current transactor stake decrease fees.
func (t *Transactor) FetchStakeDecreaseFee(chainID int64) (FeesResponse, error) {
	cachedFees := t.feeCache.getCachedFee(chainID, stakeDecreaseFeeType)
	if cachedFees != nil {
		return *cachedFees, nil
	}

	f := FeesResponse{}
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("fee/%v/stake/decrease", chainID), nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	if err == nil {
		t.feeCache.cacheFee(chainID, stakeDecreaseFeeType, f)
	}
	return f, err
}

// CombinedFeesResponse represents the combined fees response.
type CombinedFeesResponse struct {
	Current    Fees      `json:"current"`
	Last       Fees      `json:"last"`
	ServerTime time.Time `json:"server_time"`
}

// Fees represents fees for a given time frame.
type Fees struct {
	DecreaseStake *big.Int  `json:"decreaseStake"`
	Settle        *big.Int  `json:"settle"`
	Register      *big.Int  `json:"register"`
	ValidUntil    time.Time `json:"valid_until"`
}

// FetchCombinedFees fetches current transactor fees.
func (t *Transactor) FetchCombinedFees(chainID int64) (CombinedFeesResponse, error) {
	f := CombinedFeesResponse{}
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("fee/%v", chainID), nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	return f, err
}

// SettleAndRebalance requests the transactor to settle and rebalance the given channel
func (t *Transactor) SettleAndRebalance(hermesID, providerID string, promise pc.Promise) (string, error) {
	payload := PromiseSettlementRequest{
		HermesID:      hermesID,
		ProviderID:    providerID,
		ChannelID:     hex.EncodeToString(promise.ChannelID),
		Amount:        promise.Amount,
		TransactorFee: promise.Fee,
		Preimage:      hex.EncodeToString(promise.R),
		Signature:     hex.EncodeToString(promise.Signature),
		ChainID:       promise.ChainID,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle_and_rebalance", payload)
	if err != nil {
		return "", errors.Wrap(err, "failed to create settle and rebalance request")
	}
	res := SettleResponse{}
	return res.ID, t.httpClient.DoRequestAndParseResponse(req, &res)
}

func (t *Transactor) registerIdentity(endpoint string, id string, stake, fee *big.Int, beneficiary string, chainID int64) error {
	regReq, err := t.fillIdentityRegistrationRequest(id, stake, fee, beneficiary, chainID)
	if err != nil {
		return errors.Wrap(err, "failed to fill in identity request")
	}

	err = t.validateRegisterIdentityRequest(regReq)
	if err != nil {
		return errors.Wrap(err, "identity request validation failed")
	}

	req, err := requests.NewPostRequest(t.endpointAddress, endpoint, regReq)
	if err != nil {
		return errors.Wrap(err, "failed to create RegisterIdentity request")
	}

	err = t.httpClient.DoRequest(req)
	if err != nil {
		return err
	}

	// This is left as a synchronous call on purpose.
	// We need to notify registry before returning.
	t.publisher.Publish(AppTopicTransactorRegistration, regReq)

	return nil
}

type identityRegistrationRequestWithToken struct {
	IdentityRegistrationRequest
	Token string `json:"token"`
}

func (t *Transactor) registerIdentityWithReferralToken(id string, stake *big.Int, beneficiary string, token string, chainID int64) error {
	regReq, err := t.fillIdentityRegistrationRequest(id, stake, new(big.Int), beneficiary, chainID)
	if err != nil {
		return errors.Wrap(err, "failed to fill in identity request")
	}

	err = t.validateRegisterIdentityRequest(regReq)
	if err != nil {
		return errors.Wrap(err, "identity request validation failed")
	}

	r := identityRegistrationRequestWithToken{
		IdentityRegistrationRequest: regReq,
		Token:                       token,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/register/referer", r)
	if err != nil {
		return errors.Wrap(err, "failed to create RegisterIdentity request")
	}

	err = t.httpClient.DoRequest(req)
	if err != nil {
		return err
	}

	// This is left as a synchronous call on purpose.
	// We need to notify registry before returning.
	t.publisher.Publish(AppTopicTransactorRegistration, regReq)

	return nil
}

// TokenRewardResponse represents the token reward response.
type TokenRewardResponse struct {
	Reward *big.Int `json:"reward"`
}

// RegisterIdentity instructs Transactor to register identity on behalf of a client identified by 'id'
func (t *Transactor) RegisterIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error {
	if referralToken == nil {
		return t.registerIdentity("identity/register", id, stake, fee, beneficiary, chainID)
	}

	return t.registerIdentityWithReferralToken(id, stake, beneficiary, *referralToken, chainID)
}

// RegisterProviderIdentity instructs Transactor to register Provider on behalf of a client identified by 'id'
func (t *Transactor) RegisterProviderIdentity(id string, stake, fee *big.Int, beneficiary string, chainID int64, referralToken *string) error {
	if referralToken == nil {
		return t.registerIdentity("identity/register/provider", id, stake, fee, beneficiary, chainID)
	}

	return t.registerIdentityWithReferralToken(id, stake, beneficiary, *referralToken, chainID)
}

func (t *Transactor) fillIdentityRegistrationRequest(id string, stake, fee *big.Int, beneficiary string, chainID int64) (IdentityRegistrationRequest, error) {
	registry, err := t.addresser.GetRegistryAddress(chainID)
	if err != nil {
		return IdentityRegistrationRequest{}, err
	}

	hermes, err := t.addresser.GetActiveHermes(chainID)
	if err != nil {
		return IdentityRegistrationRequest{}, err
	}

	chimp, err := t.addresser.GetActiveChannelImplementation(chainID)
	if err != nil {
		return IdentityRegistrationRequest{}, err
	}

	regReq := IdentityRegistrationRequest{
		RegistryAddress: registry.Hex(),
		HermesID:        hermes.Hex(),
		Stake:           stake,
		Fee:             fee,
		Beneficiary:     beneficiary,
		ChainID:         chainID,
	}

	if regReq.Stake == nil {
		regReq.Stake = big.NewInt(0)
	}

	if regReq.Fee == nil {
		fees, err := t.FetchRegistrationFees(chainID)
		if err != nil {
			return IdentityRegistrationRequest{}, errors.Wrap(err, "could not get registration fees")
		}
		regReq.Fee = fees.Fee
	}

	if regReq.Beneficiary == "" {
		channelAddress, err := pc.GenerateChannelAddress(id, hermes.Hex(), registry.Hex(), chimp.Hex())
		if err != nil {
			return IdentityRegistrationRequest{}, errors.Wrap(err, "failed to calculate channel address")
		}

		regReq.Beneficiary = channelAddress
	}

	signer := t.signerFactory(identity.FromAddress(id))

	sig, err := t.signRegistrationRequest(signer, regReq)
	if err != nil {
		return IdentityRegistrationRequest{}, errors.Wrap(err, "failed to sign identity registration request")
	}

	signatureHex := common.Bytes2Hex(sig)
	regReq.Signature = strings.ToLower(fmt.Sprintf("0x%v", signatureHex))
	regReq.Identity = id

	return regReq, nil
}

func (t *Transactor) getReferralTokenRequest(id common.Address) (pc.ReferralTokenRequest, error) {
	signature, err := t.signerFactory(identity.FromAddress(id.Hex())).Sign(id.Bytes())
	return pc.ReferralTokenRequest{
		Identity:  id,
		Signature: hex.EncodeToString(signature.Bytes()),
	}, err
}

func (t *Transactor) validateRegisterIdentityRequest(regReq IdentityRegistrationRequest) error {
	if regReq.HermesID == "" {
		return errors.New("HermesID is required")
	}
	if regReq.RegistryAddress == "" {
		return errors.New("RegistryAddress is required")
	}
	return nil
}

func (t *Transactor) signRegistrationRequest(signer identity.Signer, regReq IdentityRegistrationRequest) ([]byte, error) {
	req := registration.Request{
		RegistryAddress: strings.ToLower(regReq.RegistryAddress),
		HermesID:        strings.ToLower(regReq.HermesID),
		Stake:           regReq.Stake,
		Fee:             regReq.Fee,
		Beneficiary:     strings.ToLower(regReq.Beneficiary),
		ChainID:         regReq.ChainID,
	}

	message := req.GetMessage()

	signature, err := signer.Sign(message)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign a registration request")
	}

	err = pc.ReformatSignatureVForBC(signature.Bytes())
	if err != nil {
		return nil, errors.Wrap(err, "signature reformat failed")
	}

	return signature.Bytes(), nil
}

// OpenChannelRequest represents the open consumer channel request body
type OpenChannelRequest struct {
	TransactorFee   *big.Int `json:"transactorFee"`
	Signature       string   `json:"signature"`
	HermesID        string   `json:"hermesID"`
	ChainID         int64    `json:"chainID"`
	RegistryAddress string   `json:"registry_address"`
}

// sign OpenChannelRequest by identity's signer
func (t *Transactor) signOpenChannelRequest(signer identity.Signer, req *OpenChannelRequest) error {
	r := registration.OpenConsumerChannelRequest{
		ChainID:         req.ChainID,
		HermesID:        req.HermesID,
		TransactorFee:   req.TransactorFee,
		RegistryAddress: req.RegistryAddress,
	}
	message := r.GetMessage()

	signature, err := signer.Sign(message)
	if err != nil {
		return fmt.Errorf("failed to sign a open channel request: %w", err)
	}

	err = pc.ReformatSignatureVForBC(signature.Bytes())
	if err != nil {
		return fmt.Errorf("signature reformat failed: %w", err)
	}

	signatureHex := common.Bytes2Hex(signature.Bytes())
	req.Signature = strings.ToLower(fmt.Sprintf("0x%v", signatureHex))

	return nil
}

// create request for open  channel
func (t *Transactor) createOpenChannelRequest(chainID int64, id, hermesID, registryAddress string) (OpenChannelRequest, error) {
	request := OpenChannelRequest{
		TransactorFee:   new(big.Int),
		HermesID:        hermesID,
		ChainID:         chainID,
		RegistryAddress: registryAddress,
	}

	signer := t.signerFactory(identity.FromAddress(id))
	err := t.signOpenChannelRequest(signer, &request)
	if err != nil {
		return request, fmt.Errorf("failed to sign open channel request: %w", err)
	}

	return request, nil
}

// OpenChannel opens payment channel for consumer for certain Hermes
func (t *Transactor) OpenChannel(chainID int64, id, hermesID, registryAddress string) error {
	endpoint := "channel/open"
	request, err := t.createOpenChannelRequest(chainID, id, hermesID, registryAddress)
	if err != nil {
		return fmt.Errorf("failed to create open channel request: %w", err)
	}

	req, err := requests.NewPostRequest(t.endpointAddress, endpoint, request)
	if err != nil {
		return fmt.Errorf("failed to do open channel request: %w", err)
	}

	return t.httpClient.DoRequest(req)
}

// ChannelStatusRequest request for channel status
type ChannelStatusRequest struct {
	Identity        string `json:"identity"`
	HermesID        string `json:"hermesID"`
	ChainID         int64  `json:"chainID"`
	RegistryAddress string `json:"registry_address"`
}

// ChannelStatus represents status of the channel
type ChannelStatus = string

const (
	// ChannelStatusNotFound channel is not opened and the request was not sent
	ChannelStatusNotFound = ChannelStatus("not_found")
	// ChannelStatusOpen channel successfully opened
	ChannelStatusOpen = ChannelStatus("open")
	// ChannelStatusFail channel open transaction fails
	ChannelStatusFail = ChannelStatus("fail")
	// ChannelStatusInProgress channel opening is in progress
	ChannelStatusInProgress = ChannelStatus("in_progress")
)

// ChannelStatusResponse represents response with channel status
type ChannelStatusResponse struct {
	Status ChannelStatus `json:"status"`
}

// ChannelStatus check the status of the channel
func (t *Transactor) ChannelStatus(chainID int64, id, hermesID, registryAddress string) (ChannelStatusResponse, error) {
	endpoint := "channel/status"
	request := ChannelStatusRequest{
		HermesID:        hermesID,
		Identity:        id,
		ChainID:         chainID,
		RegistryAddress: registryAddress,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, endpoint, request)
	if err != nil {
		return ChannelStatusResponse{}, fmt.Errorf("failed to create channel status request: %w", err)
	}

	res := ChannelStatusResponse{}
	err = t.httpClient.DoRequestAndParseResponse(req, &res)
	if err != nil {
		return ChannelStatusResponse{}, fmt.Errorf("failed to do channel status request: %w", err)
	}

	return res, nil
}

// SettleWithBeneficiaryRequest represent the request for setting new beneficiary address.
type SettleWithBeneficiaryRequest struct {
	Promise     PromiseSettlementRequest
	Beneficiary string   `json:"beneficiary"`
	Nonce       *big.Int `json:"nonce"`
	Signature   string   `json:"signature"`
	ProviderID  string   `json:"providerID"`
	ChainID     int64    `json:"chainID"`
	Registry    string   `json:"registry"`
}

// QueueResponse represents the queue response from transactor.
type QueueResponse struct {
	ID    string `json:"id"`
	Hash  string `json:"tx_hash"`
	State string `json:"state"`
	Error string `json:"error"`
}

// GetQueueStatus returns the queued transaction status from transactor.
func (t *Transactor) GetQueueStatus(ID string) (QueueResponse, error) {
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("queue/%v", ID), nil)
	if err != nil {
		return QueueResponse{}, fmt.Errorf("failed to create RegisterIdentity request %w", err)
	}
	res := QueueResponse{}
	return res, t.httpClient.DoRequestAndParseResponse(req, &res)
}

// SettleResponse represents the settle response from transactor.
type SettleResponse struct {
	ID string `json:"id"`
}

// SettleWithBeneficiary instructs Transactor to set beneficiary on behalf of a client identified by 'id'
func (t *Transactor) SettleWithBeneficiary(id, beneficiary, hermesID string, promise pc.Promise) (string, error) {
	registry, err := t.addresser.GetRegistryAddress(promise.ChainID)
	if err != nil {
		return "", err
	}

	signedReq, err := t.fillSetBeneficiaryRequest(promise.ChainID, id, beneficiary, registry.Hex())
	if err != nil {
		return "", fmt.Errorf("failed to fill in set beneficiary request: %w", err)
	}

	payload := SettleWithBeneficiaryRequest{
		Promise: PromiseSettlementRequest{
			HermesID:      hermesID,
			ChannelID:     hex.EncodeToString(promise.ChannelID),
			Amount:        promise.Amount,
			TransactorFee: promise.Fee,
			Preimage:      hex.EncodeToString(promise.R),
			Signature:     hex.EncodeToString(promise.Signature),
			ChainID:       promise.ChainID,
			ProviderID:    id,
		},
		Beneficiary: signedReq.Beneficiary,
		Nonce:       signedReq.Nonce,
		Signature:   signedReq.Signature,
		ProviderID:  id,
		ChainID:     promise.ChainID,
		Registry:    registry.Hex(),
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle_with_beneficiary", payload)
	if err != nil {
		return "", fmt.Errorf("failed to create RegisterIdentity request %w", err)
	}

	res := SettleResponse{}
	return res.ID, t.httpClient.DoRequestAndParseResponse(req, &res)
}

func (t *Transactor) fillSetBeneficiaryRequest(chainID int64, id, beneficiary, registry string) (pc.SetBeneficiaryRequest, error) {
	nonce, err := t.bc.GetLastRegistryNonce(chainID, common.HexToAddress(registry))
	if err != nil {
		return pc.SetBeneficiaryRequest{}, fmt.Errorf("failed to get last registry nonce: %w", err)
	}

	regReq := pc.SetBeneficiaryRequest{
		ChainID:     chainID,
		Registry:    registry,
		Beneficiary: strings.ToLower(beneficiary),
		Identity:    id,
		Nonce:       new(big.Int).Add(nonce, big.NewInt(1)),
	}

	signer := t.signerFactory(identity.FromAddress(id))

	sig, err := t.signSetBeneficiaryRequest(signer, regReq)
	if err != nil {
		return pc.SetBeneficiaryRequest{}, fmt.Errorf("failed to sign set beneficiary request: %w", err)
	}

	signatureHex := common.Bytes2Hex(sig)
	regReq.Signature = strings.ToLower(fmt.Sprintf("0x%v", signatureHex))

	return regReq, nil
}

func (t *Transactor) signSetBeneficiaryRequest(signer identity.Signer, req pc.SetBeneficiaryRequest) ([]byte, error) {
	message := req.GetMessage()

	signature, err := signer.Sign(message)
	if err != nil {
		return nil, fmt.Errorf("failed to sign set beneficiary request: %w", err)
	}

	err = pc.ReformatSignatureVForBC(signature.Bytes())
	if err != nil {
		return nil, fmt.Errorf("signature reformat failed: %w", err)
	}

	return signature.Bytes(), nil
}

// TransactorRegistrationEntryStatus represents the registration status.
type TransactorRegistrationEntryStatus string

const (
	// TransactorRegistrationEntryStatusCreated tells us that the registration is created.
	TransactorRegistrationEntryStatusCreated = TransactorRegistrationEntryStatus("created")
	// TransactorRegistrationEntryStatusPriceIncreased tells us that registration was requeued with an increased price.
	TransactorRegistrationEntryStatusPriceIncreased = TransactorRegistrationEntryStatus("priceIncreased")
	// TransactorRegistrationEntryStatusFailed tells us that the registration has failed.
	TransactorRegistrationEntryStatusFailed = TransactorRegistrationEntryStatus("failed")
	// TransactorRegistrationEntryStatusSucceed tells us that the registration has succeeded.
	TransactorRegistrationEntryStatusSucceed = TransactorRegistrationEntryStatus("succeed")
)

// TransactorStatusResponse represents the current registration status.
type TransactorStatusResponse struct {
	IdentityID   string                            `json:"identity_id"`
	Status       TransactorRegistrationEntryStatus `json:"status"`
	TxHash       string                            `json:"tx_hash"`
	CreatedAt    time.Time                         `json:"created_at"`
	UpdatedAt    time.Time                         `json:"updated_at"`
	BountyAmount *big.Int                          `json:"bounty_amount"`
	ChainID      int64                             `json:"chain_id"`
}

// FetchRegistrationStatus fetches current transactor registration status for given identity.
func (t *Transactor) FetchRegistrationStatus(id string) ([]TransactorStatusResponse, error) {
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("identity/%v/status", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactor registration status: %w", err)
	}

	var resp []TransactorStatusResponse
	return resp, t.httpClient.DoRequestAndParseResponse(req, &resp)
}

// SettleIntoStake requests the transactor to settle and transfer the balance to stake.
func (t *Transactor) SettleIntoStake(hermesID, providerID string, promise pc.Promise) (string, error) {
	payload := PromiseSettlementRequest{
		HermesID:      hermesID,
		ChannelID:     hex.EncodeToString(promise.ChannelID),
		Amount:        promise.Amount,
		TransactorFee: promise.Fee,
		Preimage:      hex.EncodeToString(promise.R),
		Signature:     hex.EncodeToString(promise.Signature),
		ProviderID:    providerID,
		ChainID:       promise.ChainID,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle/into_stake", payload)
	if err != nil {
		return "", errors.Wrap(err, "failed to create settle into stake request")
	}
	res := SettleResponse{}
	return res.ID, t.httpClient.DoRequestAndParseResponse(req, &res)
}

// EligibilityResponse shows if one is eligible for free registration.
type EligibilityResponse struct {
	Eligible bool `json:"eligible"`
}

// GetFreeProviderRegistrationEligibility determines if there are any free provider registrations available.
func (t *Transactor) GetFreeProviderRegistrationEligibility() (bool, error) {
	e := EligibilityResponse{}

	req, err := requests.NewGetRequest(t.endpointAddress, "identity/register/provider/eligibility", nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch registration eligibility")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &e)
	return e.Eligible, err
}

// GetFreeRegistrationEligibility determines if the identity is eligible for free registration.
func (t *Transactor) GetFreeRegistrationEligibility(identity identity.Identity) (bool, error) {
	e := EligibilityResponse{}

	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("identity/register/eligibility/%v", identity.Address), nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to fetch registration eligibility")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &e)
	return e.Eligible, err
}

// PayAndSettlePayload represents the pay and settle payload.
type PayAndSettlePayload struct {
	PromiseSettlementRequest
	Beneficiary          string `json:"beneficiary"`
	BeneficiarySignature string `json:"beneficiarySignature"`
}

// PayAndSettle requests the transactor to withdraw money into l1.
func (t *Transactor) PayAndSettle(hermesID, providerID string, promise pc.Promise, beneficiary string, beneficiarySignature string) (string, error) {
	payload := PayAndSettlePayload{
		PromiseSettlementRequest: PromiseSettlementRequest{
			HermesID:      hermesID,
			ChannelID:     hex.EncodeToString(promise.ChannelID),
			Amount:        promise.Amount,
			TransactorFee: promise.Fee,
			Preimage:      hex.EncodeToString(promise.R),
			Signature:     hex.EncodeToString(promise.Signature),
			ProviderID:    providerID,
			ChainID:       promise.ChainID,
		},
		Beneficiary:          beneficiary,
		BeneficiarySignature: beneficiarySignature,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/pay_and_settle", payload)
	if err != nil {
		return "", errors.Wrap(err, "failed to create pay and settle request")
	}
	res := SettleResponse{}
	return res.ID, t.httpClient.DoRequestAndParseResponse(req, &res)
}

// DecreaseProviderStakeRequest represents all the parameters required for decreasing provider stake.
type DecreaseProviderStakeRequest struct {
	ChannelID     string `json:"channel_id,omitempty"`
	Nonce         uint64 `json:"nonce,omitempty"`
	HermesID      string `json:"hermes_id,omitempty"`
	Amount        uint64 `json:"amount,omitempty"`
	TransactorFee uint64 `json:"transactor_fee,omitempty"`
	Signature     string `json:"signature,omitempty"`
	ChainID       int64  `json:"chain_id"`
	ProviderID    string `json:"providerID"`
}

// DecreaseStake requests the transactor to decrease stake.
func (t *Transactor) DecreaseStake(id string, chainID int64, amount, transactorFee *big.Int) error {
	payload, err := t.fillDecreaseStakeRequest(id, chainID, amount, transactorFee)
	if err != nil {
		return errors.Wrap(err, "failed to fill decrease stake request")
	}

	log.Debug().Msgf("req chid %v", payload.ChannelID)

	req, err := requests.NewPostRequest(t.endpointAddress, "stake/decrease", payload)
	if err != nil {
		return errors.Wrap(err, "failed to create decrease stake request")
	}
	return t.httpClient.DoRequest(req)
}

func (t *Transactor) fillDecreaseStakeRequest(id string, chainID int64, amount, transactorFee *big.Int) (DecreaseProviderStakeRequest, error) {
	hermes, err := t.addresser.GetActiveHermes(chainID)
	if err != nil {
		return DecreaseProviderStakeRequest{}, err
	}
	ch, err := t.bc.GetProviderChannel(chainID, hermes, common.HexToAddress(id), false)
	if err != nil {
		return DecreaseProviderStakeRequest{}, fmt.Errorf("failed to get provider channel: %w", err)
	}

	addr, err := pc.GenerateProviderChannelID(id, hermes.Hex())
	if err != nil {
		return DecreaseProviderStakeRequest{}, fmt.Errorf("failed to generate provider channel ID: %w", err)
	}

	bytes := common.FromHex(addr)
	chid := [32]byte{}
	copy(chid[:], bytes)

	req := pc.DecreaseProviderStakeRequest{
		ChannelID:     chid,
		Nonce:         ch.LastUsedNonce.Add(ch.LastUsedNonce, big.NewInt(1)),
		HermesID:      hermes,
		Amount:        amount,
		TransactorFee: transactorFee,
		ChainID:       chainID,
	}
	signer := t.signerFactory(identity.FromAddress(id))
	signature, err := signer.Sign(req.GetMessage())
	if err != nil {
		return DecreaseProviderStakeRequest{}, fmt.Errorf("failed to sign set decrease stake request: %w", err)
	}

	err = pc.ReformatSignatureVForBC(signature.Bytes())
	if err != nil {
		return DecreaseProviderStakeRequest{}, fmt.Errorf("signature reformat failed: %w", err)
	}
	signatureHex := common.Bytes2Hex(signature.Bytes())
	regReq := DecreaseProviderStakeRequest{
		Signature:     signatureHex,
		ChannelID:     common.Bytes2Hex(req.ChannelID[:]),
		Nonce:         req.Nonce.Uint64(),
		HermesID:      req.HermesID.Hex(),
		Amount:        req.Amount.Uint64(),
		TransactorFee: req.TransactorFee.Uint64(),
		ChainID:       req.ChainID,
		ProviderID:    id,
	}
	return regReq, nil
}
