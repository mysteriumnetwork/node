package openvpn

import (
	"github.com/mysterium/node/session"
	"github.com/stretchr/testify/assert"
	"testing"
)

const tlsTestKey = `
-----BEGIN OpenVPN Static key V1-----
7573bf79ebecb38d2a009d28830ecf5b0b11e27362513fe4b09b55f07054c4c7c3cebeb00bf8bb2d05cfa0f79308e762e684b931db2179e7a21618ea
869cbb5b1b9753ca05d3b87708389ccc154c9278a92964002ea888c1011fb06444088162ff6a4c1d5a8ee0ab30fd1b4dc9aaaa8c8901b426d25063cc
660d47103ff14e2cae99ca9ce28d70f927d090c144c49b3d86832c1e1c67562a6d248dff8a2583948a065015ec84d8d7bfe63385e257a6338471e2c6
7075416f4771beb0c872cc09c9ce4318fd8c9446987664f04ceeeb4e3c49f7101aa4953795014696a2f4e1cb129127fe5830627563efb127589b3693
addc15c1393f4db6c7f8d55ba598fbe5
-----END OpenVPN Static key V1-----
`

const caCertificate = `
-----BEGIN CERTIFICATE-----
MIIByDCCAW6gAwIBAgICBFcwCgYIKoZIzj0EAwIwQzELMAkGA1UEBhMCR0IxGzAZ
BgNVBAoTEk15c3Rlcm1pdW0ubmV0d29yazEXMBUGA1UECxMOTXlzdGVyaXVtIFRl
YW0wHhcNMTgwNTA4MTIwMDU5WhcNMjgwNTA4MTIwMDU5WjBDMQswCQYDVQQGEwJH
QjEbMBkGA1UEChMSTXlzdGVybWl1bS5uZXR3b3JrMRcwFQYDVQQLEw5NeXN0ZXJp
dW0gVGVhbTBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABKvoBgL5PCWlUr4PSl2j
jSXtW8ohVESWVL6l0de+Sj6dWsjELxmLAKdnwep9CcYvGE0i3Q0M24C/ZSoCREpl
8UOjUjBQMA4GA1UdDwEB/wQEAwIChDAdBgNVHSUEFjAUBggrBgEFBQcDAgYIKwYB
BQUHAwEwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ4EBwQFAQIDBAUwCgYIKoZIzj0E
AwIDSAAwRQIhAKLOIPprhU7CCyFG52J8FmyzwBJjcwHu+ZzGFrdfwEKKAiB7xkYM
YFcPCscvdnZ1U8hTUaREZmDB2w9eaGyCM4YXAg==
-----END CERTIFICATE-----
`

func TestValidatorReturnsNilErrorOnValidVPNConfig(t *testing.T) {
	vpnConfig := session.VPNConfig{
		"1.2.3.4",
		10999,
		"tcp",
		tlsTestKey,
		caCertificate,
	}
	assert.NoError(t, NewDefaultValidator().IsValid(vpnConfig))
}

func TestIPv6AreNotAllowed(t *testing.T) {
	vpnConfig := session.VPNConfig{RemoteIP: "2001:db8:85a3::8a2e:370:7334"}
	assert.Error(t, validIPFormat(vpnConfig))
}

func TestUnknownProtocolIsNotAllowed(t *testing.T) {
	vpnConfig := session.VPNConfig{RemoteProtocol: "fake_protocol"}
	assert.Error(t, validProtocol(vpnConfig))
}

func TestPortOutOfRangeIsNotAllowed(t *testing.T) {
	vpnConfig := session.VPNConfig{RemotePort: -1}
	assert.Error(t, validPort(vpnConfig))
}

func TestTLSPresharedKeyIsValid(t *testing.T) {
	vpnConfig := session.VPNConfig{TLSPresharedKey: tlsTestKey}
	assert.NoError(t, validTLSPresharedKey(vpnConfig))
}

func TestCACertificateIsValid(t *testing.T) {
	vpnConfig := session.VPNConfig{CACertificate: caCertificate}
	assert.NoError(t, validCACertificate(vpnConfig))
}
