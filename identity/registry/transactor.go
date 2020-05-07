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
)

// AppTopicTransactorRegistration represents the registration topic to which events regarding registration attempts on transactor will occur
const AppTopicTransactorRegistration = "transactor_identity_registration"

// AppTopicTransactorTopUp represents the top up topic to which events regarding top up attempts are sent.
const AppTopicTransactorTopUp = "transactor_top_up"

type channelProvider interface {
	GetProviderChannel(accountantAddress common.Address, addressToCheck common.Address, pending bool) (client.ProviderChannel, error)
}

// Transactor allows for convenient calls to the transactor service
type Transactor struct {
	httpClient            *requests.HTTPClient
	endpointAddress       string
	signerFactory         identity.SignerFactory
	registryAddress       string
	accountantID          string
	channelImplementation string
	publisher             eventbus.Publisher
	bc                    channelProvider
}

// NewTransactor creates and returns new Transactor instance
func NewTransactor(httpClient *requests.HTTPClient, endpointAddress, registryAddress, accountantID, channelImplementation string, signerFactory identity.SignerFactory, publisher eventbus.Publisher, bc channelProvider) *Transactor {
	return &Transactor{
		httpClient:            httpClient,
		endpointAddress:       endpointAddress,
		signerFactory:         signerFactory,
		registryAddress:       registryAddress,
		accountantID:          accountantID,
		channelImplementation: channelImplementation,
		publisher:             publisher,
		bc:                    bc,
	}
}

// FeesResponse represents fees applied by Transactor
type FeesResponse struct {
	Fee        uint64    `json:"fee"`
	ValidUntil time.Time `json:"valid_until"`
}

// IsValid returns false if the fee has already expired and should be re-requested
func (fr FeesResponse) IsValid() bool {
	return time.Now().After(fr.ValidUntil)
}

// IdentityRegistrationRequestDTO represents the identity registration user input parameters
// swagger:model IdentityRegistrationRequestDTO
type IdentityRegistrationRequestDTO struct {
	// Stake is used by Provider, default 0
	Stake uint64 `json:"stake,omitempty"`
	// Cache out address for Provider
	Beneficiary string `json:"beneficiary,omitempty"`
	// Fee: negotiated fee with transactor
	Fee uint64 `json:"fee,omitempty"`
}

// TopUpRequest represents the myst top up request
// swagger:model TopUpRequestDTO
type TopUpRequest struct {
	// Identity to top up with myst
	Identity string `json:"identity"`
}

// IdentityRegistrationRequest represents the identity registration request body
type IdentityRegistrationRequest struct {
	RegistryAddress string `json:"registryAddress"`
	AccountantID    string `json:"accountantID"`
	// Stake is used by Provider, default 0
	Stake uint64 `json:"stake"`
	// Fee: negotiated fee with transactor
	Fee uint64 `json:"fee"`
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
	AccountantID  string `json:"accountantID"`
	ChannelID     string `json:"channelID"`
	Amount        uint64 `json:"amount"`
	TransactorFee uint64 `json:"fee"`
	Preimage      string `json:"preimage"`
	Signature     string `json:"signature"`
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

// TopUp requests a myst topup for testing purposes.
func (t *Transactor) TopUp(id string) error {
	channelAddress, err := pc.GenerateChannelAddress(id, t.accountantID, t.registryAddress, t.channelImplementation)
	if err != nil {
		return errors.Wrap(err, "failed to calculate channel address")
	}

	payload := TopUpRequest{
		Identity: channelAddress,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "topup", payload)
	if err != nil {
		return errors.Wrap(err, "failed to create TopUp request")
	}

	// This is left as a synchronous call on purpose.
	t.publisher.Publish(AppTopicTransactorTopUp, id)

	return t.httpClient.DoRequest(req)
}

// SettleAndRebalance requests the transactor to settle and rebalance the given channel
func (t *Transactor) SettleAndRebalance(accountantID string, promise pc.Promise) error {
	payload := PromiseSettlementRequest{
		AccountantID:  accountantID,
		ChannelID:     hex.EncodeToString(promise.ChannelID),
		Amount:        promise.Amount,
		TransactorFee: promise.Fee,
		Preimage:      hex.EncodeToString(promise.R),
		Signature:     hex.EncodeToString(promise.Signature),
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle_and_rebalance", payload)
	if err != nil {
		return errors.Wrap(err, "failed to create TopUp request")
	}
	return t.httpClient.DoRequest(req)
}

// RegisterIdentity instructs Transactor to register identity on behalf of a client identified by 'id'
func (t *Transactor) RegisterIdentity(id string, regReqDTO *IdentityRegistrationRequestDTO) error {
	regReq, err := t.fillIdentityRegistrationRequest(id, *regReqDTO)
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

	// This is left as a synchronous call on purpose.
	// We need to notify registry before returning.
	t.publisher.Publish(AppTopicTransactorRegistration, regReq)

	return t.httpClient.DoRequest(req)
}

func (t *Transactor) fillIdentityRegistrationRequest(id string, regReqDTO IdentityRegistrationRequestDTO) (IdentityRegistrationRequest, error) {
	regReq := IdentityRegistrationRequest{
		RegistryAddress: t.registryAddress,
		AccountantID:    t.accountantID,
		Stake:           regReqDTO.Stake,
		Fee:             regReqDTO.Fee,
		Beneficiary:     regReqDTO.Beneficiary,
	}

	if regReq.Beneficiary == "" {
		channelAddress, err := pc.GenerateChannelAddress(id, t.accountantID, t.registryAddress, t.channelImplementation)
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

func (t *Transactor) validateRegisterIdentityRequest(regReq IdentityRegistrationRequest) error {
	if regReq.AccountantID == "" {
		return errors.New("AccountantID is required")
	}
	if regReq.RegistryAddress == "" {
		return errors.New("RegistryAddress is required")
	}
	return nil
}

func (t *Transactor) signRegistrationRequest(signer identity.Signer, regReq IdentityRegistrationRequest) ([]byte, error) {
	req := registration.Request{
		RegistryAddress: strings.ToLower(regReq.RegistryAddress),
		AccountantID:    strings.ToLower(regReq.AccountantID),
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

type SettleWithBeneficiaryRequest struct {
	Promise     PromiseSettlementRequest
	Beneficiary string `json:"beneficiary"`
	Nonce       uint64 `json:"nonce"`
	Signature   string `json:"signature"`
}

// SetBeneficiary instructs Transactor to set beneficiary on behalf of a client identified by 'id'
func (t *Transactor) SettleWithBeneficiary(id, beneficiary, accountantID string, promise pc.Promise) error {
	signedReq, err := t.fillSetBeneficiaryRequest(id, beneficiary)
	if err != nil {
		return fmt.Errorf("failed to fill in set beneficiary request: %w", err)
	}

	payload := SettleWithBeneficiaryRequest{
		Promise: PromiseSettlementRequest{
			AccountantID:  accountantID,
			ChannelID:     hex.EncodeToString(promise.ChannelID),
			Amount:        promise.Amount,
			TransactorFee: promise.Fee,
			Preimage:      hex.EncodeToString(promise.R),
			Signature:     hex.EncodeToString(promise.Signature),
		},
		Beneficiary: signedReq.Beneficiary,
		Nonce:       signedReq.Nonce,
		Signature:   signedReq.Signature,
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "identity/settle_with_beneficiary", payload)
	if err != nil {
		return fmt.Errorf("failed to create RegisterIdentity request %w", err)
	}

	return t.httpClient.DoRequest(req)
}

func (t *Transactor) fillSetBeneficiaryRequest(id, beneficiary string) (pc.SetBeneficiaryRequest, error) {
	ch, err := t.bc.GetProviderChannel(common.HexToAddress(t.accountantID), common.HexToAddress(id), false)
	if err != nil {
		return pc.SetBeneficiaryRequest{}, fmt.Errorf("failed to get provider channel: %w", err)
	}

	addr, err := pc.GenerateProviderChannelID(id, t.accountantID)
	if err != nil {
		return pc.SetBeneficiaryRequest{}, fmt.Errorf("failed to generate provider channel ID: %w", err)
	}

	regReq := pc.SetBeneficiaryRequest{
		Beneficiary: strings.ToLower(beneficiary),
		ChannelID:   strings.ToLower(addr),
		Nonce:       ch.LastUsedNonce.Uint64() + 1,
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
