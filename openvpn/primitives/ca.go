package primitives

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"os"
)

// createCA creates Certificate Authority certificate and private key
func (p *SecurityPrimitives) createCA(serviceLocation dto.Location) (*x509.Certificate, error) {

	if err := p.cleanup(p.CACertPath); err != nil {
		return nil, err
	}

	if err := p.cleanup(p.CAKeyPath); err != nil {
		return nil, err
	}

	log.Info(logPrefix, "Create CA (", p.CACertPath, ", ", p.CAKeyPath, ")")

	ca := newCACert(serviceLocation)

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

	if err := writeCertAsPEM(p.CACertPath, p.caBytes); err != nil {
		return nil, err
	}

	if err := writeKeyAsPEM(p.CAKeyPath, p.caPrivateKey); err != nil {
		return nil, err
	}

	return ca, nil
}

// createCert creates certificate and private key signed by given CA certificate
func (p *SecurityPrimitives) createCert(parentCA *x509.Certificate, server bool, serviceLocation dto.Location, providerID identity.Identity) error {

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

	cert := newServerCert(extUsage, serviceLocation, providerID)

	var err error
	p.serverPrivateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, parentCA, &p.serverPrivateKey.PublicKey, p.caPrivateKey)
	if err != nil {
		log.Error(logPrefix, "certificate creation failed: ", err)
		return err
	}
	p.serverCertBytes = certBytes

	if err := writeCertAsPEM(p.ServerCertPath, p.serverCertBytes); err != nil {
		return err
	}

	if err := writeKeyAsPEM(p.ServerKeyPath, p.serverPrivateKey); err != nil {
		return err
	}

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

func writeCertAsPEM(certPath string, certBytes []byte) error {
	certOut, err := os.Create(certPath)
	if err != nil {
		log.Error(logPrefix, "failed to open "+certPath+" for writing: %s", err)
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()
	log.Debug(logPrefix, "written "+certPath)

	return nil
}

func writeKeyAsPEM(keyPath string, privateKey *ecdsa.PrivateKey) error {
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Debug(logPrefix, "failed to open "+keyPath+" for writing:", err)
		return err
	}
	defer keyOut.Close()

	block, err := pemBlockForKey(privateKey)
	if err != nil {
		return err
	}

	if err := pem.Encode(keyOut, block); err != nil {
		return err
	}

	log.Debug(logPrefix, "written "+keyPath)
	return nil
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
