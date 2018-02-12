package primitives

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestCreateTLSCryptKeyFileExists(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)
	sp.createTLSCryptKey()

	if _, err := os.Stat(sp.TLSCryptKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.TLSCryptKeyPath)
	}
}

func TestTLSCryptKeyFileContentsAreValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)
	sp.createTLSCryptKey()

	keyFile, err := os.OpenFile(sp.TLSCryptKeyPath, os.O_RDONLY, 0600)
	if err != nil {
		t.Errorf("failed to open %s", sp.TLSCryptKeyPath)
	}
	// 256 bytes after hex encode becomes larger
	fileLen := 512 + len("-----BEGIN OpenVPN Static key V1-----\n") + len("\n-----END OpenVPN Static key V1-----\n")
	keyBytes := make([]byte, fileLen)
	contentLen, _ := keyFile.Read(keyBytes)

	if fileLen != contentLen {
		t.Errorf("content lengh %s should be %s", contentLen, fileLen)
		assert.Equal(t, contentLen, fileLen)
	}

	assert.Contains(t, string(keyBytes), "-----BEGIN OpenVPN Static key V1-----")
	assert.Contains(t, string(keyBytes), "-----END OpenVPN Static key V1-----")
}
