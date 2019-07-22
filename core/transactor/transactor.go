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

package transactor

import (
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysteriumnetwork/payments/registration"
	"github.com/pkg/errors"

	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/requests"
)

type transactor struct {
	http            requests.HTTPTransport
	endpointAddress string
	registryAddress string
	accountantID    string
	signerFactory   identity.SignerFactory
	regReq          *IdentityRegistrationRequest
}

// NewTransactor creates and returns new Transactor instance
func NewTransactor(endpointAddress, registryAddress, accountantID string, signerFactory identity.SignerFactory) *transactor {
	return &transactor{
		http:            requests.NewHTTPClient(20 * time.Second),
		endpointAddress: endpointAddress,
		signerFactory:   signerFactory,
		regReq:          &IdentityRegistrationRequest{RegistryAddress: registryAddress, AccountantID: accountantID},
	}
}

// Fees represents fees applied by Transactor
type Fees struct {
	Transaction  uint64 `json:"transaction"`
	Registration uint64 `json:"registration"`
}

// IdentityRegistrationRequestDTO represents the identity registration user input parameters
type IdentityRegistrationRequestDTO struct {
	// Stake is used by Provider, default 0
	Stake uint64 `json:"stake,omitempty"`
	// Cache out address for Provider
	Beneficiary string `json:"beneficiary,omitempty"`
	// Fee: negotiated fee with transactor
	Fee uint64 `json:"fee,omitempty"`
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

//  FetchFees fetches current transactor fees
func (t *transactor) FetchFees() (Fees, error) {
	f := Fees{}

	req, err := requests.NewGetRequest(t.endpointAddress, "fee/register", nil)
	if err != nil {
		return f, errors.Wrap(err, "failed to fetch transactor fees")
	}

	err = t.http.DoRequestAndParseResponse(req, &f)
	if err != nil {
		return f, err
	}

	return f, nil
}

// RegisterIdentity instructs Transactor to register identity on behalf of a client identified by 'id'
func (t *transactor) RegisterIdentity(id string, regReqDTO *IdentityRegistrationRequestDTO) error {

	err := t.fillIdentityRegistrationRequest(id, regReqDTO)
	if err != nil {
		return errors.Wrap(err, "failed to fill in identity request")
	}

	err = t.validateRegisterIdentityRequest()
	if err != nil {
		return errors.Wrap(err, "identity request validation failed")
	}

	req, err := requests.NewPostRequest(t.endpointAddress, "/identity/register", t.regReq)
	if err != nil {
		return errors.Wrap(err, "identity request to Transactor failed")
	}

	err = t.http.DoRequest(req)
	if err != nil {
		return err
	}
	return nil
}

func (t *transactor) fillIdentityRegistrationRequest(id string, regReqDTO *IdentityRegistrationRequestDTO) error {
	t.regReq.Stake = regReqDTO.Stake
	t.regReq.Beneficiary = regReqDTO.Beneficiary
	t.regReq.Fee = regReqDTO.Fee

	signer := t.signerFactory(identity.FromAddress(id))

	sig, err := t.signRegistrationRequest(signer)
	if err != nil {
		return errors.Wrap(err, "failed to sign identity registration request")
	}

	signatureHex := common.Bytes2Hex(sig)
	t.regReq.Signature = strings.ToLower(fmt.Sprintf("0x%v", signatureHex))

	t.regReq.Identity = id

	return nil
}

func (t *transactor) validateRegisterIdentityRequest() error {
	if t.regReq.AccountantID == "" {
		return errors.New("AccountantID is required")
	}
	if t.regReq.RegistryAddress == "" {
		return errors.New("RegistryAddress is required")
	}
	return nil
}

func (t *transactor) signRegistrationRequest(signer identity.Signer) ([]byte, error) {
	req := registration.Request{
		RegistryAddress: strings.ToLower(t.regReq.RegistryAddress),
		AccountantID:    strings.ToLower(t.regReq.AccountantID),
		Stake:           t.regReq.Stake,
		Fee:             t.regReq.Fee,
		Beneficiary:     strings.ToLower(t.regReq.Beneficiary),
	}

	message := req.GetMessage()
	hash := crypto.Keccak256Hash(message)

	signature, err := signer.Sign(hash.Bytes())
	return signature.Bytes(), err
}
