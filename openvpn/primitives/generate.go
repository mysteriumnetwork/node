package primitives

import (
	"crypto/ecdsa"
	"crypto/x509"
	"fmt"
	log "github.com/cihub/seelog"
	"os"
	"path/filepath"
)

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

func (sp *SecurityPrimitives) Init() {
	sp.cleanupDir()

	sp.mkDir(sp.directory)
	sp.mkDir(filepath.Join(sp.directory, "ca"))
	sp.mkDir(filepath.Join(sp.directory, "server"))
	sp.mkDir(filepath.Join(sp.directory, "clients"))
	sp.mkDir(filepath.Join(sp.directory, "common"))
}

func NewOpenVPNSecPrimitives() *SecurityPrimitives {
	dir := "certs"
	return &SecurityPrimitives{
		dir,
		filepath.Join(dir, "ca", "ca.crt"),
		filepath.Join(dir, "ca", "ca.key"),
		filepath.Join(dir, "server", "server.crt"),
		filepath.Join(dir, "server", "server.key"),
		filepath.Join("bin", "tls", "crl.pem"),
		filepath.Join(dir, "ta.key"),
		nil,
		nil,
		nil,
		nil,
	}
}

func (sp *SecurityPrimitives) CACert() string {
	return sp.caCertPath
}

func (sp *SecurityPrimitives) CrlPEM() string {
	return sp.crlPEMPath
}

func (sp *SecurityPrimitives) TLSCryptKey() string {
	return sp.tlsCryptKeyPath
}

func (sp *SecurityPrimitives) ServerCert() string {
	return sp.serverCertPath
}

func (sp *SecurityPrimitives) ServerKey() string {
	return sp.serverKeyPath
}

func (sp *SecurityPrimitives) cleanupDir() {
	if _, err := os.Stat(sp.directory); os.IsNotExist(err) {
		return
	}
	err := RemoveContents(sp.directory)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func RemoveContents(dir string) error {
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

func (sp *SecurityPrimitives) Generate() {
	var err error

	var ca *x509.Certificate

	sp.Init()

	if ca, err = sp.CreateCA(); err != nil {
		log.Info("CreateCA failed: ", err)
		return
	}

	if err = sp.CreateCert(ca, true); err != nil {
		log.Info("CreateCert failed: ", err)
		return
	}

	if err = sp.CreateTA(sp.tlsCryptKeyPath); err != nil {
		log.Info("CreateTA failed: ", err)
		return
	}
}
