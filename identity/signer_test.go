package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"github.com/ethereum/go-ethereum/accounts/keystore"
)

func TestSigningMessageWithUnlockedAccount(t *testing.T) {
	ks := keystore.NewKeyStore("test_data", keystore.StandardScryptN, keystore.StandardScryptP)

	manager := NewIdentityManager(ks)
	err := manager.Unlock("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68", "")
	assert.NoError(t, err)

	signer := NewSigner(ks, FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68"))
	message := []byte("MystVpnSessionId:Boop!")
	signature, err := signer.Sign([]byte(message))
	signatureBase64 := signature.Base64()
	t.Logf("signature in base64: %s", signatureBase64)
	assert.NoError(t, err)
	assert.Equal(
		t,
		SignatureBase64("V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE="),
		signature,
	)
}
