package identity

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	originalSignerID = FromAddress("0x53a835143c0ef3bbcbfa796d7eb738ca7dd28f68")
	hijackedSignerID = FromAddress("0xded9913d38bfe94845b9e21fd32f43d0240e2f34")
)

func TestAuthenticate_WhenSignatureIsCorrect(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, originalSignerID, signerID, "Original signer should be extracted")
}

func TestAuthenticate_WhenPrefixedMessageSignatureIsCorrect(t *testing.T) {
	message := []byte("MystVpnSessionId:Boop!")
	signature := SignatureBase64("57a89f9af2ee013fa16ed2c15ffd319b70b469fcb0c5321dc351ea2e6038a27a709a26c76f1561979d06af769151932ac355b191f48855d8696c296e1c604ab201")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.NoError(t, err)
	assert.Exactly(t, originalSignerID, signerID, "Original signer should be extracted")
}

func TestAuthenticate_WhenSignatureIsEmpty(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.EqualError(t, err, "empty signature")
	assert.Exactly(t, Identity{}, signerID)
}

func TestAuthenticate_WhenSignatureIsMalformed(t *testing.T) {
	message := []byte("Boop!")
	signature := SignatureHex("7369676e6564")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.EqualError(t, err, "invalid signature length")
	assert.Exactly(t, Identity{}, signerID)
}

func TestAuthenticate_WhenMessageIsChanged(t *testing.T) {
	message := []byte("Boop changed!")
	signature := SignatureHex("1f89542f406b2d638fe09cd9912d0b8c0b5ebb4aef67d52ab046973e34fb430a1953576cd19d140eddb099aea34b2985fbd99e716d3b2f96a964141fdb84b32000")

	extractor := &extractor{}
	signerID, err := extractor.Extract(message, signature)
	assert.NoError(t, err)
	assert.NotEqual(t, originalSignerID, signerID, "Original signer should not be extracted")
	assert.Exactly(t, hijackedSignerID, signerID, "Another signer extracted")
}
