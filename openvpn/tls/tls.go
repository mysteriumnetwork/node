package tls

import (
	"crypto/x509"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
)

type Primitives struct {
	CertificateAuthority *CertificateAuthority
	ServerCertificate    *CertificateKeyPair
	PresharedKey         *TLSPresharedKey
}

func NewTLSPrimitives(serviceLocation dto.Location, serviceProviderID identity.Identity) (*Primitives, error) {

	key, err := createTLSCryptKey()
	if err != nil {
		return nil, err
	}

	ca, err := CreateAuthority(newCACert(serviceLocation))
	if err != nil {
		return nil, err
	}

	server, err := ca.CreateDerived(newServerCert(x509.ExtKeyUsageServerAuth, serviceLocation, serviceProviderID))
	if err != nil {
		return nil, err
	}

	return &Primitives{
		CertificateAuthority: ca,
		ServerCertificate:    server,
		PresharedKey:         &key,
	}, nil
}
