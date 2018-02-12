package primitives

import (
	"crypto/ecdsa"
	"crypto/x509"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"os"
	"path/filepath"
)

// SecurityPrimitives describes security primitives
type SecurityPrimitives struct {
	CACertPath, CAKeyPath         string
	ServerCertPath, ServerKeyPath string
	TLSCryptKeyPath               string
	directory                     string
	caBytes                       []byte
	caPrivateKey                  *ecdsa.PrivateKey
	serverCertBytes               []byte
	serverPrivateKey              *ecdsa.PrivateKey
}

// GenerateOpenVPNSecPrimitives returns pre-initialized SecurityPrimitives structure
func GenerateOpenVPNSecPrimitives(directoryRuntime string, serviceLocation dto.Location, providerID identity.Identity) (*SecurityPrimitives, error) {
	sp := newOpenVPNSecPrimitives(directoryRuntime)
	err := sp.generateAll(serviceLocation, providerID)
	return sp, err
}

const logPrefix = "[config-openvpn] "

const (
	caCertFile      = "ca.crt"
	tlsCryptKeyFile = "tc.key"
)

// These two are needed for client

// CACertPath returns path to CA certificate
func CACertPath(directoryRuntime string) string {
	return filepath.Join(directoryRuntime, caCertFile)
}

// TLSCryptKeyPath returns path to TLS crypt file
func TLSCryptKeyPath(directoryRuntime string) string {
	return filepath.Join(directoryRuntime, tlsCryptKeyFile)
}

func newOpenVPNSecPrimitives(directoryRuntime string) *SecurityPrimitives {
	return &SecurityPrimitives{
		filepath.Join(directoryRuntime, caCertFile),
		filepath.Join(directoryRuntime, "ca.key"),
		filepath.Join(directoryRuntime, "server.crt"),
		filepath.Join(directoryRuntime, "server.key"),
		filepath.Join(directoryRuntime, tlsCryptKeyFile),
		directoryRuntime,
		nil,
		nil,
		nil,
		nil,
	}
}

// remove previously generated files
func (p *SecurityPrimitives) cleanup(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil
	}

	err := os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

// generateAll generates required security primitives
func (p *SecurityPrimitives) generateAll(serviceLocation dto.Location, providerID identity.Identity) error {
	var err error
	var ca *x509.Certificate

	if ca, err = p.createCA(serviceLocation); err != nil {
		log.Info(logPrefix, "createCA failed: ", err)
		return err
	}

	if err = p.createCert(ca, true, serviceLocation, providerID); err != nil {
		log.Info(logPrefix, "createCert failed: ", err)
		return err
	}

	if err = p.checkCertificate(); err != nil {
		log.Info(logPrefix, "checkCertificate failed: ", err)
		return err
	}

	if err = p.createTLSCryptKey(); err != nil {
		log.Info(logPrefix, "createTLSCryptKey failed: ", err)
		return err
	}

	return nil
}
