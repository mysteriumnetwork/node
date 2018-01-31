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
	"github.com/enceve/crypto/dh"
	"math/big"
	"os"
	"time"
)

func (p *SecurityPrimitives) CreateCA() (*x509.Certificate, error) {
	log.Info("Create CA (", p.caCert, ", ", p.caKey, ")")

	/*
		ca := &x509.Certificate{
			SerialNumber: big.NewInt(1653),
			Subject: pkix.Name{
				Country:            []string{"China"},
				Organization:       []string{"Yjwt"},
				OrganizationalUnit: []string{"YjwtU"},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			SubjectKeyId:          []byte{1, 2, 3, 4, 5},
			BasicConstraintsValid: true,
			IsCA:        true,
			ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		}
	*/
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

	certOut, err := os.Create(p.caCert)
	if err != nil {
		log.Info("failed to open "+p.caCert+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: p.caBytes})
	certOut.Close()
	log.Info("written " + p.caCert + "\n")

	keyOut, err := os.OpenFile(p.caKey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+p.caKey+" for writing:", err)
		return nil, err
	}
	pem.Encode(keyOut, pemBlockForKey(p.caPrivateKey))
	keyOut.Close()
	log.Info("written " + p.caKey + "\n")

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

func (p *SecurityPrimitives) caCertPath() string {
	return p.caCert
}

func (p *SecurityPrimitives) caKeyPath() string {
	return p.caKey
}

func (p *SecurityPrimitives) CreateCert(parentCA *x509.Certificate, server bool) error {
	log.Info("Create certificate (", p.serverCert, ", ", p.serverKey, ")")

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

	/*
		cert := &x509.Certificate{
			SerialNumber: big.NewInt(1658),
			Subject: pkix.Name{
				Country:            []string{"China"},
				Organization:       []string{"Fuck"},
				OrganizationalUnit: []string{"FuckU"},
			},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().AddDate(10, 0, 0),
			SubjectKeyId: []byte{1, 2, 3, 4, 6},
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		}
	*/
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
	certOut, err := os.Create(p.serverCert)
	if err != nil {
		log.Info("failed to open "+p.serverCert+" for writing: %s", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()
	log.Info("written " + p.serverCert + "\n")

	// server key in PEM
	keyOut, err := os.OpenFile(p.serverKey, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+p.serverKey+" for writing:", err)
		return err
	}
	pem.Encode(keyOut, pemBlockForKey(p.serverPrivateKey))
	keyOut.Close()
	log.Info("written " + p.serverKey + "\n")

	return nil
}

// DH params not needed for EC?
// https://forums.openvpn.net/viewtopic.php?t=23227
// but still mandatory for openvpn for fallback cases:
// https://community.openvpn.net/openvpn/ticket/410
func (p *SecurityPrimitives) CreateDH() error {
	group := dh.RFC3526_2048()
	_, _, err := group.GenerateKey(rand.Reader)
	if err != nil {
		log.Info("Failed to generate private / public key pair")
	}
	return err
}

func (p *SecurityPrimitives) checkCertificate(caBytes []byte, certBytes []byte) {
	ca, _ := x509.ParseCertificate(caBytes)
	cert, _ := x509.ParseCertificate(certBytes)
	err := cert.CheckSignatureFrom(ca)
	if err != nil {
		log.Info("failed to check signature ", err)
	}
}
