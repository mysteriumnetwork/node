package identity

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSigningMessageWithUnlockedAccount(t *testing.T) {
	identity := FromAddress("0x1e35193c8cadAA15b43B05ae3D882C91F49BB0Aa")
	ks := getIdentityTestKeystore()
	ks.Unlock(identityToAccount(identity), "test_passphrase")

	signer := keystoreSigner{ks, identityToAccount(identity)}
	sig, err := signer.Sign([]byte("Boop!"))
	assert.NoError(t, err)

	assert.Equal(
		t,
		"40d313f383460ccfe632176f39102b4a1644f8cbf95c15ccac7c5858e66a1e545efd9ac530fe24f606cb9b33e2b6c124f420c2e3c7324f3b64fc1a7c3d4fc6d001",
		hex.EncodeToString(sig),
	)
}
