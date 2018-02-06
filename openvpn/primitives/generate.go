package primitives

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	log "github.com/cihub/seelog"
	"os"
	"path/filepath"
)

// SecurityPrimitives describes security primitives
type SecurityPrimitives struct {
	directory        string
	caCertPath       string
	caKeyPath        string
	serverCertPath   string
	serverKeyPath    string
	crlPEMPath       string
	tlsCryptKeyPath  string
	caBytes          []byte
	caPrivateKey     *ecdsa.PrivateKey
	serverCertBytes  []byte
	serverPrivateKey *ecdsa.PrivateKey
}

func (sp *SecurityPrimitives) mkDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Debug("Creating dir (" + dir + ")")
		os.Mkdir(dir, 0600)
	}
}

const (
	genDir    = "certs"
	caDir     = "ca"
	serverDir = "server"
	clientDir = "clients"
	commonDir = "common"
)

func (sp *SecurityPrimitives) init() {
	sp.cleanupDir()

	sp.mkDir(sp.directory)
	sp.mkDir(filepath.Join(sp.directory, caDir))
	sp.mkDir(filepath.Join(sp.directory, serverDir))
	sp.mkDir(filepath.Join(sp.directory, clientDir))
	sp.mkDir(filepath.Join(sp.directory, commonDir))
}

// NewOpenVPNSecPrimitives returns pre-initialized SecurityPrimitives structure
func NewOpenVPNSecPrimitives() *SecurityPrimitives {
	return &SecurityPrimitives{
		genDir,
		filepath.Join(genDir, caDir, "ca.crt"),
		filepath.Join(genDir, caDir, "ca.key"),
		filepath.Join(genDir, serverDir, "server.crt"),
		filepath.Join(genDir, serverDir, "server.key"),
		filepath.Join("bin", "tls", "crl.pem"),
		filepath.Join(genDir, commonDir, "ta.key"),
		nil,
		nil,
		nil,
		nil,
	}
}

// CACert returns CA certificate file path
func (sp *SecurityPrimitives) CACert() string {
	return sp.caCertPath
}

// CrlPEM returns CRL file path
func (sp *SecurityPrimitives) CrlPEM() string {
	return sp.crlPEMPath
}

// TLSCryptKey returns TLS crypt file path
func (sp *SecurityPrimitives) TLSCryptKey() string {
	return sp.tlsCryptKeyPath
}

// ServerCert returns server TLS certificate
func (sp *SecurityPrimitives) ServerCert() string {
	return sp.serverCertPath
}

// ServerKey returns server private key
func (sp *SecurityPrimitives) ServerKey() string {
	return sp.serverKeyPath
}

func (sp *SecurityPrimitives) cleanupDir() {
	if _, err := os.Stat(sp.directory); os.IsNotExist(err) {
		return
	}
	err := removeContents(sp.directory)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// Generate generates required security primitives
func (sp *SecurityPrimitives) Generate() {
	var err error

	var ca *x509.Certificate

	sp.init()

	if ca, err = sp.CreateCA(); err != nil {
		log.Info("CreateCA failed: ", err)
		return
	}

	if err = sp.CreateCert(ca, true); err != nil {
		log.Info("CreateCert failed: ", err)
		return
	}

	if err = sp.CreateTLSCryptKey(sp.tlsCryptKeyPath); err != nil {
		log.Info("CreateTLSCryptKey failed: ", err)
		return
	}
}
