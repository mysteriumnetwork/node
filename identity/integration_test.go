package identity

import (
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_UnlockAndSignAndVerify(t *testing.T) {
	ks := keystore.NewKeyStore("test_data", keystore.StandardScryptN, keystore.StandardScryptP)

	manager := NewIdentityManager(ks)
	err := manager.Unlock("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68", "")
	assert.NoError(t, err)

	signer := NewSigner(ks, FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	signature, err := signer.Sign([]byte("Boop!"))
	assert.NoError(t, err)
	assert.Exactly(
		t,
		"1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000",
		signature,
	)

	verifier := NewVerifier(FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	assert.True(t, verifier.Verify([]byte("Boop!"), signature))
}
