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
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
	"github.com/mysteriumnetwork/payments/client"
	pc "github.com/mysteriumnetwork/payments/crypto"
	"github.com/mysteriumnetwork/payments/registration"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// AppTopicTransactorRegistration represents the registration topic to which events regarding registration attempts on transactor will occur
const AppTopicTransactorRegistration = "transactor_identity_registration"

type channelProvider interface {
	GetProviderChannel(hermesAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
}

// Transactor allows for convenient calls to the transactor service
type Transactor struct {
	httpClient            *requests.HTTPClient
	endpointAddress       string
	signerFactory         identity.SignerFactory
	registryAddress       string
	hermesID              string
	channelImplementation string
	publisher             eventbus.Publisher
	bc                    channelProvider
}

// NewTransactor creates and returns new Transactor instance
func NewTransactor(httpClient *requests.HTTPClient, endpointAddress, registryAddress, hermesID, channelImplementation string, signerFactory identity.SignerFactory, publisher eventbus.Publisher, bc channelProvider) *Transactor {
	return &Transactor{
		httpClient:            httpClient,
		endpointAddress:       endpointAddress,
		signerFactory:         signerFactory,
		registryAddress:       registryAddress,
		hermesID:              hermesID,
		channelImplementation: channelImplementation,
		publisher:             publisher,
		bc:                    bc,
	}
}

// FeesResponse represents fees applied by Transactor
type FeesResponse struct {
	Fee        *big.Int  `json:"fee"`
	ValidUntil time.Time `json:"valid_until"`
}

// IsValid returns false if the fee has already expired and should be re-requested
func (fr FeesResponse) IsValid() bool {
	return time.Now().After(fr.ValidUntil)
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
}

// FetchRegistrationFees fetches current transactor registration fees
func (t *Transactor) FetchRegistrationFees() (FeesResponse, error) {
	f := FeesResponse{}

	req, err := requests.NewGetRequest(t.endpointAddress, "fee/register", nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	return f, err
}

// FetchSettleFees fetches current transactor settlement fees
func (t *Transactor) FetchSettleFees() (FeesResponse, error) {
	f := FeesResponse{}

	req, err := requests.NewGetRequest(t.endpointAddress, "fee/settle", nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	return f, err
}

// FetchStakeDecreaseFee fetches current transactor stake decrease fees.
func (t *Transactor) FetchStakeDecreaseFee() (FeesResponse, error) {
	f := FeesResponse{}

	req, err := requests.NewGetRequest(t.endpointAddress, "fee/stake/decrease", nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	return f, err
}

// SettleAndRebalance requests the transactor to settle and rebalance the given channel
func (t *Transactor) SettleAndRebalance(hermesID, providerID string, promise pc.Promise) error {
	payload := PromiseSettlementRequest{
		HermesID:      hermesID,
		ProviderID:    providerID,
		ChannelID:     hex.EncodeToString(promise.ChannelID),
		Amount:        promise.Amount,
		TransactorFee: promise.Fee,
		Preimage:      hex.EncodeToString(promise.R),
		Signature:     hex.EncodeToString(promise.Signature),
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle_and_rebalance", payload)
	if err != nil {
		return errors.Wrap(err, "failed to create settle and rebalance request")
	}
	return t.httpClient.DoRequest(req)
}

func (t *Transactor) registerIdentity(id string, stake, fee *big.Int, beneficiary string) error {
	regReq, err := t.fillIdentityRegistrationRequest(id, stake, fee, beneficiary)
	if err != nil {
		return errors.Wrap(err, "failed to fill in identity request")
	}

	err = t.validateRegisterIdentityRequest(regReq)
	if err != nil {
		return errors.Wrap(err, "identity request validation failed")
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/register", regReq)
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

func (t *Transactor) registerIdentityWithReferralToken(id string, stake *big.Int, beneficiary string, token string) error {
	regReq, err := t.fillIdentityRegistrationRequest(id, stake, new(big.Int), beneficiary)
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

// GetTokenReward returns the reward that is issued for the given token.
func (t *Transactor) GetTokenReward(token string) (TokenRewardResponse, error) {
	f := TokenRewardResponse{}
	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("referal/%v/reward", token), nil)
	if err != nil {
		return f, fmt.Errorf("failed to fetch transactor fees %w", err)
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	return f, err
}

// RegisterIdentity instructs Transactor to register identity on behalf of a client identified by 'id'
func (t *Transactor) RegisterIdentity(id string, stake, fee *big.Int, beneficiary string, referralToken *string) error {
	if referralToken == nil {
		return t.registerIdentity(id, stake, fee, beneficiary)
	}

	return t.registerIdentityWithReferralToken(id, stake, beneficiary, *referralToken)
}

func (t *Transactor) fillIdentityRegistrationRequest(id string, stake, fee *big.Int, beneficiary string) (IdentityRegistrationRequest, error) {
	regReq := IdentityRegistrationRequest{
		RegistryAddress: t.registryAddress,
		HermesID:        t.hermesID,
		Stake:           stake,
		Fee:             fee,
		Beneficiary:     beneficiary,
	}

	if regReq.Stake == nil {
		regReq.Stake = big.NewInt(0)
	}

	if regReq.Fee == nil {
		fees, err := t.FetchRegistrationFees()
		if err != nil {
			return IdentityRegistrationRequest{}, errors.Wrap(err, "could not get registration fees")
		}
		regReq.Fee = fees.Fee
	}

	if regReq.Beneficiary == "" {
		channelAddress, err := pc.GenerateChannelAddress(id, t.hermesID, t.registryAddress, t.channelImplementation)
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

// GetReferralToken returns the referral token
func (t *Transactor) GetReferralToken(id common.Address) (string, error) {
	req, err := t.getReferralTokenRequest(id)
	if err != nil {
		return "", err
	}

	request, err := requests.NewPostRequest(t.endpointAddress, "rp/tokens/request", req)
	if err != nil {
		return "", fmt.Errorf("failed to create referral token request %w", err)
	}

	var resp struct {
		Token string `json:"token"`
	}
	err = t.httpClient.DoRequestAndParseResponse(request, &resp)
	return resp.Token, err
}

func (t *Transactor) getReferralTokenRequest(id common.Address) (pc.ReferralTokenRequest, error) {
	signature, err := t.signerFactory(identity.FromAddress(id.Hex())).Sign(id.Bytes())
	return pc.ReferralTokenRequest{
		Identity:  id,
		Signature: hex.EncodeToString(signature.Bytes()),
	}, err
}

// CheckIfRegistrationBountyEligible determines if the identity is eligible for registration bounty
func (t *Transactor) CheckIfRegistrationBountyEligible(identity identity.Identity) (bool, error) {
	signer := t.signerFactory(identity)
	message := common.HexToAddress(identity.Address)
	signature, err := signer.Sign(message.Bytes())
	if err != nil {
		return false, err
	}

	req := pc.ReferralTokenRequest{
		Identity:  common.HexToAddress(identity.Address),
		Signature: hex.EncodeToString(signature.Bytes()),
	}

	request, err := requests.NewPostRequest(t.endpointAddress, "identity/register/bounty", req)
	if err != nil {
		return false, fmt.Errorf("failed to create RegisterIdentity request %w", err)
	}

	resp, err := t.httpClient.Do(request)
	if err != nil {
		return false, fmt.Errorf("failed to check bounty status %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("got unexpected status from bounty check %v", resp.StatusCode)
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

// SettleWithBeneficiaryRequest represent the request for setting new beneficiary address.
type SettleWithBeneficiaryRequest struct {
	Promise     PromiseSettlementRequest
	Beneficiary string   `json:"beneficiary"`
	Nonce       *big.Int `json:"nonce"`
	Signature   string   `json:"signature"`
	ProviderID  string   `json:"providerID"`
}

// SettleWithBeneficiary instructs Transactor to set beneficiary on behalf of a client identified by 'id'
func (t *Transactor) SettleWithBeneficiary(id, beneficiary, hermesID string, promise pc.Promise) error {
	signedReq, err := t.fillSetBeneficiaryRequest(promise.ChainID, id, beneficiary)
	if err != nil {
		return fmt.Errorf("failed to fill in set beneficiary request: %w", err)
	}

	payload := SettleWithBeneficiaryRequest{
		Promise: PromiseSettlementRequest{
			HermesID:      hermesID,
			ChannelID:     hex.EncodeToString(promise.ChannelID),
			Amount:        promise.Amount,
			TransactorFee: promise.Fee,
			Preimage:      hex.EncodeToString(promise.R),
			Signature:     hex.EncodeToString(promise.Signature),
		},
		Beneficiary: signedReq.Beneficiary,
		Nonce:       signedReq.Nonce,
		Signature:   signedReq.Signature,
		ProviderID:  id,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle_with_beneficiary", payload)
	if err != nil {
		return fmt.Errorf("failed to create RegisterIdentity request %w", err)
	}

	return t.httpClient.DoRequest(req)
}

func (t *Transactor) fillSetBeneficiaryRequest(chainID int64, id, beneficiary string) (pc.SetBeneficiaryRequest, error) {
	ch, err := t.bc.GetProviderChannel(common.HexToAddress(t.hermesID), common.HexToAddress(id), false)
	if err != nil {
		return pc.SetBeneficiaryRequest{}, fmt.Errorf("failed to get provider channel: %w", err)
	}

	regReq := pc.SetBeneficiaryRequest{
		ChainID:     chainID,
		Beneficiary: strings.ToLower(beneficiary),
		Identity:    id,
		Nonce:       new(big.Int).Add(ch.LastUsedNonce, big.NewInt(1)),
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
}

// FetchRegistrationStatus fetches current transactor registration status for given identity.
func (t *Transactor) FetchRegistrationStatus(id string) (TransactorStatusResponse, error) {
	f := TransactorStatusResponse{}

	req, err := requests.NewGetRequest(t.endpointAddress, fmt.Sprintf("identity/%v/status", id), nil)
	if err != nil {
		return f, fmt.Errorf("failed to fetch transactor registration status: %w", err)
	}

	err = t.httpClient.DoRequestAndParseResponse(req, &f)
	return f, err
}

// SettleIntoStake requests the transactor to settle and transfer the balance to stake.
func (t *Transactor) SettleIntoStake(hermesID, providerID string, promise pc.Promise) error {
	payload := PromiseSettlementRequest{
		HermesID:      hermesID,
		ChannelID:     hex.EncodeToString(promise.ChannelID),
		Amount:        promise.Amount,
		TransactorFee: promise.Fee,
		Preimage:      hex.EncodeToString(promise.R),
		Signature:     hex.EncodeToString(promise.Signature),
		ProviderID:    providerID,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle/into_stake", payload)
	if err != nil {
		return errors.Wrap(err, "failed to create settle into stake request")
	}
	return t.httpClient.DoRequest(req)
}

// DecreaseProviderStakeRequest represents all the parameters required for decreasing provider stake.
type DecreaseProviderStakeRequest struct {
	ChannelID     string `json:"channel_id,omitempty"`
	Nonce         uint64 `json:"nonce,omitempty"`
	HermesID      string `json:"hermes_id,omitempty"`
	Amount        uint64 `json:"amount,omitempty"`
	TransactorFee uint64 `json:"transactor_fee,omitempty"`
	Signature     string `json:"signature,omitempty"`
}

// DecreaseStake requests the transactor to decrease stake.
func (t *Transactor) DecreaseStake(id string, amount, transactorFee *big.Int) error {
	payload, err := t.fillDecreaseStakeRequest(id, amount, transactorFee)
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

func (t *Transactor) fillDecreaseStakeRequest(id string, amount, transactorFee *big.Int) (DecreaseProviderStakeRequest, error) {
	ch, err := t.bc.GetProviderChannel(common.HexToAddress(t.hermesID), common.HexToAddress(id), false)
	if err != nil {
		return DecreaseProviderStakeRequest{}, fmt.Errorf("failed to get provider channel: %w", err)
	}

	addr, err := pc.GenerateProviderChannelID(id, t.hermesID)
	if err != nil {
		return DecreaseProviderStakeRequest{}, fmt.Errorf("failed to generate provider channel ID: %w", err)
	}

	bytes := common.FromHex(addr)
	chid := [32]byte{}
	copy(chid[:], bytes)

	req := pc.DecreaseProviderStakeRequest{
		ChannelID:     chid,
		Nonce:         ch.LastUsedNonce.Add(ch.LastUsedNonce, big.NewInt(1)),
		HermesID:      common.HexToAddress(t.hermesID),
		Amount:        amount,
		TransactorFee: transactorFee,
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
	}
	return regReq, nil
}
