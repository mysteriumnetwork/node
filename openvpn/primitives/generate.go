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
	CACertPath       string
	CAKeyPath        string
	ServerCertPath   string
	ServerKeyPath    string
	CRLPEMPath       string
	TLSCryptKeyPath  string
	caBytes          []byte
	caPrivateKey     *ecdsa.PrivateKey
	serverCertBytes  []byte
	serverPrivateKey *ecdsa.PrivateKey
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}

func (sp *SecurityPrimitives) mkDir(dir string) {
	if e, err := exists(dir); !e {
		log.Debug("Creating dir (" + dir + ")")
		os.Mkdir(dir, 0700)
	} else {
		log.Errorf("Error creating directory", err)
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

	err = os.RemoveAll(dir)
	if err != nil {
		return err
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

	if err = sp.CheckCertificate(); err != nil {
		log.Info("CheckCertificate failed: ", err)
		return
	}

	if err = sp.CreateTLSCryptKey(); err != nil {
		log.Info("CreateTLSCryptKey failed: ", err)
		return
	}
}
