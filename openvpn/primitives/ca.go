package primitives

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	log "github.com/cihub/seelog"
	"math/big"
	"os"
	"time"
)

// createCA creates Certificate Authority certificate and private key
func (p *SecurityPrimitives) createCA() (*x509.Certificate, error) {

	if err := p.cleanup(p.CACertPath); err != nil {
		return nil, err
	}

	if err := p.cleanup(p.CAKeyPath); err != nil {
		return nil, err
	}

	log.Info(logPrefix, "Create CA (", p.CACertPath, ", ", p.CAKeyPath, ")")

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
		log.Error(logPrefix, "failed to generate private key: %s", err)
		return nil, err
	}

	p.caBytes, err = x509.CreateCertificate(rand.Reader, ca, ca, &p.caPrivateKey.PublicKey, p.caPrivateKey)
	if err != nil {
		log.Error(logPrefix, "create ca failed: ", err)
		return nil, err
	}

	certOut, err := os.Create(p.CACertPath)
	if err != nil {
		log.Error(logPrefix, "failed to open "+p.CACertPath+" for writing: %s", err)
		return nil, err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: p.caBytes})
	certOut.Close()
	log.Debug(logPrefix, "written "+p.CACertPath)

	keyOut, err := os.OpenFile(p.CAKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Debug(logPrefix, "failed to open "+p.CAKeyPath+" for writing:", err)
		return nil, err
	}
	defer keyOut.Close()

	block, err := pemBlockForKey(p.caPrivateKey)
	if err != nil {
		return nil, err
	}

	if err := pem.Encode(keyOut, block); err != nil {
		return nil, err
	}

	log.Debug(logPrefix, "written "+p.CAKeyPath)

	return ca, nil
}

func pemBlockForKey(priv interface{}) (*pem.Block, error) {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, log.Error(logPrefix, "unable to marshal ECDSA private key: %s", err)

		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, log.Error(logPrefix, "unable to marshal given key: type unknown")
	}
}

// createCert creates certificate and private key signed by given CA certificate
func (p *SecurityPrimitives) createCert(parentCA *x509.Certificate, server bool) error {

	if err := p.cleanup(p.ServerCertPath); err != nil {
		return err
	}

	if err := p.cleanup(p.ServerKeyPath); err != nil {
		return err
	}

	log.Info(logPrefix, "Create certificate (", p.ServerCertPath, ", ", p.ServerKeyPath, ")")

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
		log.Error(logPrefix, "certificate creation failed: ", err)
		return err
	}
	p.serverCertBytes = certBytes

	// cert in PEM
	certOut, err := os.Create(p.ServerCertPath)
	if err != nil {
		log.Error(logPrefix, "failed to open "+p.ServerCertPath+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()
	log.Debug(logPrefix, "written "+p.ServerCertPath)

	// key in PEM
	keyOut, err := os.OpenFile(p.ServerKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info(logPrefix, "failed to open "+p.ServerKeyPath+" for writing:", err)
		return err
	}
	defer keyOut.Close()

	block, err := pemBlockForKey(p.serverPrivateKey)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, block); err != nil {
		return err
	}

	log.Debug(logPrefix, "written "+p.ServerKeyPath)

	return nil
}

// checkCertificate checks if generated certificate signature is from parentCA
func (p *SecurityPrimitives) checkCertificate() error {
	ca, err := x509.ParseCertificate(p.caBytes)
	if err != nil {
		return log.Errorf(logPrefix, "failed to parse CA certificate: ", err)
	}

	cert, err := x509.ParseCertificate(p.serverCertBytes)
	if err != nil {
		return log.Errorf(logPrefix, "failed to parse server certificate: ", err)
	}

	err = cert.CheckSignatureFrom(ca)
	if err != nil {
		return log.Errorf(logPrefix, "failed to check signature: ", err)
	}
	return nil
}
