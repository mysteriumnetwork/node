package primitives

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

const caCertPEM = `
-----BEGIN CERTIFICATE-----
MIIB1zCCAXygAwIBAgICBnUwCgYIKoZIzj0EAwIwSjESMBAGA1UEBhMJR2licmFs
dGFyMRswGQYDVQQKExJNeXN0ZXJtaXVtLm5ldHdvcmsxFzAVBgNVBAsTDk15c3Rl
cml1bSBUZWFtMB4XDTE4MDIwNzEyNDgyNVoXDTI4MDIwNzEyNDgyNVowSjESMBAG
A1UEBhMJR2licmFsdGFyMRswGQYDVQQKExJNeXN0ZXJtaXVtLm5ldHdvcmsxFzAV
BgNVBAsTDk15c3Rlcml1bSBUZWFtMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE
KFCZQeHkUFOL9KxfN7+iWamLuqEk4wR42UW1hGM5w01742sg7s1HHjA6UZc4AnnG
ZlqRef3Tt2k23FkOtym0x6NSMFAwDgYDVR0PAQH/BAQDAgKEMB0GA1UdJQQWMBQG
CCsGAQUFBwMCBggrBgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MA4GA1UdDgQHBAUB
AgMEBTAKBggqhkjOPQQDAgNJADBGAiEAxlWQLTo0/cppxk7hIpP/KjOduewEvFSE
NnQwQOC6wygCIQDInwAbZi3doTIOysiTPHAv9rAcG5YM3leTwMwaYIs7fQ==
-----END CERTIFICATE-----`

const caKeyPEM = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKKFvIv3xWInkkwWJf9oRfyECKkbMJ4RS7YxjwWIBwnEoAoGCCqGSM49
AwEHoUQDQgAEKFCZQeHkUFOL9KxfN7+iWamLuqEk4wR42UW1hGM5w01742sg7s1H
HjA6UZc4AnnGZlqRef3Tt2k23FkOtym0xw==
-----END EC PRIVATE KEY-----`

func TestCreateCAFilesExists(t *testing.T) {
	sp := newOpenVPNSecPrimitives(runDir)
	sp.createCA()

	if _, err := os.Stat(sp.CACertPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.CACertPath)
	}

	if _, err := os.Stat(sp.CAKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.CAKeyPath)
	}
}

func TestCreateCertFilesExists(t *testing.T) {
	sp := newOpenVPNSecPrimitives(runDir)

	// get parent ca
	block, _ := pem.Decode([]byte(caCertPEM))
	ca, _ := x509.ParseCertificate(block.Bytes)

	// get ca key
	keyBlock, _ := pem.Decode([]byte(caKeyPEM))
	key, _ := x509.ParseECPrivateKey(keyBlock.Bytes)
	sp.caBytes = block.Bytes
	sp.caPrivateKey = key

	// generate sever cert / key
	sp.createCert(ca, true)

	if _, err := os.Stat(sp.ServerCertPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.ServerCertPath)
	}

	if _, err := os.Stat(sp.ServerKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.ServerKeyPath)
	}
}

func TestServerCertFileIsValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(runDir)

	// get parent ca
	block, _ := pem.Decode([]byte(caCertPEM))
	ca, _ := x509.ParseCertificate(block.Bytes)

	// get ca key
	keyBlock, _ := pem.Decode([]byte(caKeyPEM))
	key, _ := x509.ParseECPrivateKey(keyBlock.Bytes)
	sp.caBytes = block.Bytes
	sp.caPrivateKey = key

	// generate sever cert / key
	sp.createCert(ca, true)

	if err := sp.checkCertificate(); err != nil {
		t.Errorf("certificate should be valid %s", err)
	}
}

func TestCACertFileIsNotValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(runDir)

	// get parent ca
	block, _ := pem.Decode([]byte(caCertPEM))

	// generate own ca
	ownCA, _ := sp.createCA()

	// generate sever cert / key on ownCA
	sp.createCert(ownCA, true)

	// substitude with different ca
	sp.caBytes = block.Bytes

	if err := sp.checkCertificate(); err == nil {
		t.Errorf("CA certificate should be invalid %s", err)
	}
}

func TestServerCertFileIsNotValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(runDir)

	// get parent ca
	block, _ := pem.Decode([]byte(caCertPEM))
	sp.caBytes = block.Bytes

	// set bad server cert
	sp.serverCertBytes = []byte{0, 0, 0, 0, 0}

	if err := sp.checkCertificate(); err == nil {
		t.Errorf("Server certificate should be invalid %s", err)
	}
}
