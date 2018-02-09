package primitives

import (
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

func newCACert() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Country:            []string{"Gibraltar"},
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
func newServerCert(extUsage x509.ExtKeyUsage) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1658),
		Subject: pkix.Name{
			Country:            []string{"Germany"},
			CommonName:         "Mysterium Node",
			Organization:       []string{"User Company"},
			OrganizationalUnit: []string{"User Company dev team"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{extUsage},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
}
