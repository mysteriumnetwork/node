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

package validators

import (
	"crypto/ecdsa"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/node/session/promise/model"
	"github.com/stretchr/testify/assert"
)

var payerKey, _ = crypto.ToECDSA(common.FromHex("0x0b4eef4e99796ebfffe5046488525cd906ebc87b30f86ca6bd21b19dc2b319db"))
var payerSigner = NewPrivateKeySigner(payerKey)
var payerAddress = crypto.PubkeyToAddress(payerKey.PublicKey)

type privateKeySigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewPrivateKeySigner(key *ecdsa.PrivateKey) *privateKeySigner {
	return &privateKeySigner{
		privateKey: key,
	}
}

func (pkh *privateKeySigner) Sign(data []byte) (identity.Signature, error) {
	sig, err := crypto.Sign(crypto.Keccak256(data), pkh.privateKey)
	return identity.SignatureBytes(sig), err
}

func TestPromiseSignatureValidatorReturnsTrueForReceivedValidMessage(t *testing.T) {
	serviceConsumer := common.HexToAddress("0x1122334455")
	serviceProvider := common.HexToAddress("0x1122334455")

	//payer agrees to transfer 1000 tokens to service provaider on behalf of service consumer
	issuedPromise, err := model.SignByPayer(
		&model.Promise{
			Extra: promise.ExtraData{
				ConsumerAddress: serviceConsumer,
			},
			Receiver: serviceProvider,
			Amount:   1000,
			SeqNo:    10,
		},
		payerSigner,
	)
	assert.NoError(t, err)

	//a message is sent over the network
	promiseMessage := promise.Message{
		Amount:     uint64(issuedPromise.Amount),
		SequenceID: uint64(issuedPromise.SeqNo),
		Signature:  hexutil.Encode(issuedPromise.IssuerSignature),
	}

	//at the other end promise signature validator checks, that promise signature is valid
	validator := IssuedPromiseValidator{
		consumer: serviceConsumer,
		receiver: serviceProvider,
		issuer:   payerAddress,
	}

	assert.True(t, validator.Validate(promiseMessage))
}
