package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSigningMessageWithUnlockedAccount(t *testing.T) {
	message := []byte("Boop!")

	keystore := &keyStoreFake{}
	signer := keystoreSigner{keystore, addressToAccount("0x0000000000000000000000000000000000000000")}

	signature, err := signer.Sign(message)
	assert.NoError(t, err)
	assert.Equal(t, messageHash(message), keystore.LastHash)
	assert.Equal(t, SignatureHex("7369676e6564"), signature)
}
