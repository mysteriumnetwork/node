package primitives

import (
	"crypto/x509"
	"encoding/pem"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"os"
	"testing"
)

const fakeCACertPEM = `
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

const fakeCAKeyPEM = `
-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIKKFvIv3xWInkkwWJf9oRfyECKkbMJ4RS7YxjwWIBwnEoAoGCCqGSM49
AwEHoUQDQgAEKFCZQeHkUFOL9KxfN7+iWamLuqEk4wR42UW1hGM5w01742sg7s1H
HjA6UZc4AnnGZlqRef3Tt2k23FkOtym0xw==
-----END EC PRIVATE KEY-----`

const fakeServerCertPEM = `
-----BEGIN CERTIFICATE-----
MIIB5TCCAYugAwIBAgICBnowCgYIKoZIzj0EAwIwSjESMBAGA1UEBhMJR2licmFs
dGFyMRswGQYDVQQKExJNeXN0ZXJtaXVtLm5ldHdvcmsxFzAVBgNVBAsTDk15c3Rl
cml1bSBUZWFtMB4XDTE4MDIwOTE0MTY0OFoXDTE5MDIwOTE0MTY0OFowYjEQMA4G
A1UEBhMHR2VybWFueTEVMBMGA1UEChMMVXNlciBDb21wYW55MR4wHAYDVQQLExVV
c2VyIENvbXBhbnkgZGV2IHRlYW0xFzAVBgNVBAMTDk15c3Rlcml1bSBOb2RlMFkw
EwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEbsTGtfXi1xp7T6O+pXNyopM2kZmTGwL/
dh/TETR0U9rXzwfLxBRTdqBYjECpbnZh4PtMTHbXWnIUg6s+gXXlvKNJMEcwDgYD
VR0PAQH/BAQDAgKEMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA4GA1UdDgQHBAUBAgME
BjAQBgNVHSMECTAHgAUBAgMEBTAKBggqhkjOPQQDAgNIADBFAiA7/yzrQ8V+8k3f
wu7vPrmMXzdQ8sPSPtUsaQR3MSu7dQIhAP3NIutHqk/HTCoN52P/TG4fWgsbjr+v
vmJFhfSZ1ztB
-----END CERTIFICATE-----`

var fakeServiceLocation = dto.Location{"GB", "", ""}
var fakeProviderID = identity.Identity{"some fake identity "}

func TestCreateCAFilesExists(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)
	sp.createCA(fakeServiceLocation)

	if _, err := os.Stat(sp.CACertPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.CACertPath)
	}

	if _, err := os.Stat(sp.CAKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.CAKeyPath)
	}
}

func TestCreateCertFilesExists(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)

	// get parent ca
	block, _ := pem.Decode([]byte(fakeCACertPEM))
	ca, _ := x509.ParseCertificate(block.Bytes)

	// get ca key
	keyBlock, _ := pem.Decode([]byte(fakeCAKeyPEM))
	key, _ := x509.ParseECPrivateKey(keyBlock.Bytes)
	sp.caBytes = block.Bytes
	sp.caPrivateKey = key

	// generate server cert / key
	sp.createCert(ca, true, fakeServiceLocation, fakeProviderID)

	if _, err := os.Stat(sp.ServerCertPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.ServerCertPath)
	}

	if _, err := os.Stat(sp.ServerKeyPath); os.IsNotExist(err) {
		t.Errorf("file %s should exist", sp.ServerKeyPath)
	}
}

func TestServerCertFileIsValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)

	// get parent ca
	block, _ := pem.Decode([]byte(fakeCACertPEM))
	ca, _ := x509.ParseCertificate(block.Bytes)

	// get ca key
	keyBlock, _ := pem.Decode([]byte(fakeCAKeyPEM))
	key, _ := x509.ParseECPrivateKey(keyBlock.Bytes)
	sp.caBytes = block.Bytes
	sp.caPrivateKey = key

	// generate server cert / key
	sp.createCert(ca, true, fakeServiceLocation, fakeProviderID)

	if err := sp.checkCertificate(); err != nil {
		t.Errorf("certificate should be valid %s", err)
	}
}

func TestCACertFileIsNotValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)

	// get parent CA
	block, _ := pem.Decode([]byte(fakeCACertPEM))
	sp.caBytes = block.Bytes

	// get certificate signed with different CA
	block, _ = pem.Decode([]byte(fakeServerCertPEM))
	sp.serverCertBytes = block.Bytes

	if err := sp.checkCertificate(); err == nil {
		t.Errorf("CA certificate should be invalid %s", err)
	}
}

func TestServerCertFileIsNotValid(t *testing.T) {
	sp := newOpenVPNSecPrimitives(fakeRunDir)

	// get parent ca
	block, _ := pem.Decode([]byte(fakeCACertPEM))
	sp.caBytes = block.Bytes

	// set bad server cert
	sp.serverCertBytes = []byte{0, 0, 0, 0, 0}

	if err := sp.checkCertificate(); err == nil {
		t.Errorf("Server certificate should be invalid %s", err)
	}
}
