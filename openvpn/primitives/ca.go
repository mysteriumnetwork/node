package primitives

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	log "github.com/cihub/seelog"
	"math/big"
	"os"
	"time"
)

// CreateCA creates Certificate Authority certificate and private key
func (p *SecurityPrimitives) CreateCA() (*x509.Certificate, error) {
	log.Info("Create CA (", p.CACertPath, ", ", p.CAKeyPath, ")")

	ca := &x509.Certificate{
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
	var err error
	p.caPrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Info("failed to generate private key: %s", err)
		return nil, err
	}

	p.caBytes, err = x509.CreateCertificate(rand.Reader, ca, ca, &p.caPrivateKey.PublicKey, p.caPrivateKey)
	if err != nil {
		log.Info("create ca failed: ", err)
		return nil, err
	}

	certOut, err := os.Create(p.CACertPath)
	if err != nil {
		log.Info("failed to open "+p.CACertPath+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: p.caBytes})
	certOut.Close()
	log.Info("written " + p.CACertPath)

	keyOut, err := os.OpenFile(p.CAKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+p.CAKeyPath+" for writing:", err)
		return nil, err
	}
	pem.Encode(keyOut, pemBlockForKey(p.caPrivateKey))
	keyOut.Close()
	log.Info("written " + p.CAKeyPath)

	return ca, nil
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

// CreateCert creates certificate and private key signed by given CA certificate
func (p *SecurityPrimitives) CreateCert(parentCA *x509.Certificate, server bool) error {
	log.Info("Create certificate (", p.ServerCertPath, ", ", p.ServerKeyPath, ")")

	extUsage := x509.ExtKeyUsageClientAuth

	if server {
		extUsage = x509.ExtKeyUsageServerAuth
	}

	cert := &x509.Certificate{
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

	var err error
	p.serverPrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parentCA, &p.serverPrivateKey.PublicKey, p.caPrivateKey)
	if err != nil {
		log.Info("certificate creation failed: ", err)
		return err
	}
	p.serverCertBytes = certBytes

	// cert in PEM
	certOut, err := os.Create(p.ServerCertPath)
	if err != nil {
		log.Info("failed to open "+p.ServerCertPath+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()
	log.Debug("written " + p.ServerCertPath)

	// key in PEM
	keyOut, err := os.OpenFile(p.ServerKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+p.ServerKeyPath+" for writing:", err)
		return err
	}
	pem.Encode(keyOut, pemBlockForKey(p.serverPrivateKey))
	keyOut.Close()
	log.Debug("written " + p.ServerKeyPath)

	return nil
}

// CheckCertificate checks if generated certificate signature is from parentCA
func (p *SecurityPrimitives) CheckCertificate() error {
	ca, err := x509.ParseCertificate(p.caBytes)
	if err != nil {
		return log.Errorf("failed to parse CA certificate: ", err)
	}

	cert, err := x509.ParseCertificate(p.serverCertBytes)
	if err != nil {
		return log.Errorf("failed to parse server certificate: ", err)
	}

	err = cert.CheckSignatureFrom(ca)
	if err != nil {
		return log.Errorf("failed to check signature: ", err)
	}
	return nil
}
