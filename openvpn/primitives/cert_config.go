package primitives

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"math/big"
	"time"
)

func newCACert(serviceLocation dto.Location) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1111),
		Subject: pkix.Name{
			Country:            []string{serviceLocation.Country},
			Organization:       []string{"Mystermium.network"},
			OrganizationalUnit: []string{"Mysterium Team"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		SubjectKeyId:          []byte{1, 2, 3, 4, 5},
		BasicConstraintsValid: true,
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
}
func newServerCert(extUsage x509.ExtKeyUsage, serviceLocation dto.Location, providerID identity.Identity) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(2222),
		Subject: pkix.Name{
			Country:            []string{serviceLocation.Country},
			CommonName:         providerID.Address,
			Organization:       []string{"Mysterium node operator company"},
			OrganizationalUnit: []string{"Node operator team"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{extUsage},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
}
