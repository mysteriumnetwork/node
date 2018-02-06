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

func (p *SecurityPrimitives) CreateCA() (*x509.Certificate, error) {
	log.Info("Create CA (", p.caCertPath, ", ", p.caKeyPath, ")")

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
	//pri, err := rsa.GenerateKey(rand.Reader, 1024)
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

	certOut, err := os.Create(p.caCertPath)
	if err != nil {
		log.Info("failed to open "+p.caCertPath+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: p.caBytes})
	certOut.Close()
	log.Info("written " + p.caCertPath + "\n")

	keyOut, err := os.OpenFile(p.caKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+p.caKeyPath+" for writing:", err)
		return nil, err
	}
	pem.Encode(keyOut, pemBlockForKey(p.caPrivateKey))
	keyOut.Close()
	log.Info("written " + p.caKeyPath + "\n")

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

func (p *SecurityPrimitives) CreateCert(parentCA *x509.Certificate, server bool) error {
	log.Info("Create certificate (", p.serverCertPath, ", ", p.serverKeyPath, ")")

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
		//ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		ExtKeyUsage: []x509.ExtKeyUsage{extUsage},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	//pri, _ := rsa.GenerateKey(rand.Reader, 1024)
	var err error
	p.serverPrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parentCA, &p.serverPrivateKey.PublicKey, p.caPrivateKey)
	if err != nil {
		log.Info("certificate creation failed: ", err)
		return err
	}
	p.serverCertBytes = certBytes
	p.checkCertificate(p.caBytes, p.serverCertBytes)

	// server cert in PEM
	certOut, err := os.Create(p.serverCertPath)
	if err != nil {
		log.Info("failed to open "+p.serverCertPath+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()
	log.Debug("written " + p.serverCertPath + "\n")

	// server key in PEM
	keyOut, err := os.OpenFile(p.serverKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+p.serverKeyPath+" for writing:", err)
		return err
	}
	pem.Encode(keyOut, pemBlockForKey(p.serverPrivateKey))
	keyOut.Close()
	log.Debug("written " + p.serverKeyPath + "\n")

	return nil
}

func (p *SecurityPrimitives) checkCertificate(caBytes []byte, certBytes []byte) {
	ca, _ := x509.ParseCertificate(caBytes)
	cert, _ := x509.ParseCertificate(certBytes)
	err := cert.CheckSignatureFrom(ca)
	if err != nil {
		log.Info("failed to check signature ", err)
	}
}
