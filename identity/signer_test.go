package identity

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func TestSigningMessageWithUnlockedAccount(t *testing.T) {
	identity := FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68")
	ks := keystore.NewKeyStore(
		"test_data",
		keystore.StandardScryptN,
		keystore.StandardScryptP,
	)
	signer := keystoreSigner{ks, identityToAccount(identity)}
	sig, err := signer.Sign([]byte("Boop!"))
	assert.NoError(t, err)

	assert.Equal(
		t,
		"1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000",
		hex.EncodeToString(sig),
	)
}
