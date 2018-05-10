package tls

import (
	"crypto/x509"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

var fakeServiceLocation = dto.Location{"GB", "", ""}
var fakeProviderID = identity.Identity{"some fake identity "}

func TestCertificateAuthorityIsCreatedAndCertCanBeSerialized(t *testing.T) {
	_, err := CreateAuthority(newCACert(fakeServiceLocation))
	assert.NoError(t, err)
}

func TestServerCertificateIsCreatedAndBothCertAndKeyCanBeSerialized(t *testing.T) {
	ca, err := CreateAuthority(newCACert(fakeServiceLocation))
	assert.NoError(t, err)
	_, err = ca.CreateDerived(newServerCert(x509.ExtKeyUsageServerAuth, fakeServiceLocation, fakeProviderID))
	assert.NoError(t, err)
}
