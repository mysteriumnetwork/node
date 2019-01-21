package validators

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"

	"github.com/ethereum/go-ethereum/common"
	"github.com/mysteriumnetwork/node/session/promise"
	"github.com/mysteriumnetwork/payments/promises"
	"github.com/mysteriumnetwork/payments/test_utils"

	"github.com/ethereum/go-ethereum/crypto"
)

var payerKey, _ = crypto.ToECDSA(common.FromHex("0x0b4eef4e99796ebfffe5046488525cd906ebc87b30f86ca6bd21b19dc2b319db"))
var payerSigner = test_utils.NewPrivateKeySigner(payerKey)
var payerAddress = crypto.PubkeyToAddress(payerKey.PublicKey)

func TestPromiseSignatureValidatorReturnsTrueForReceivedValidMessage(t *testing.T) {
	serviceConsumer := common.HexToAddress("0x1122334455")
	serviceProvider := common.HexToAddress("0x1122334455")

	//payer agrees to transfer 1000 tokens to service provaider on behalf of service consumer
	issuedPromise, err := promises.SignByPayer(
		&promises.Promise{
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
	promiseMessage := promise.PromiseMessage{
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
