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
	caCert           string
	caKey            string
	serverCert       string
	serverKey        string
	dhPEM            string
	crlPEM           string
	taKey            string
	caBytes          []byte
	caPrivateKey     *ecdsa.PrivateKey
	serverCertBytes  []byte
	serverPrivateKey *ecdsa.PrivateKey
}

func (sp *SecurityPrimitives) mkdir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Debug("Creating dir (" + dir + ")")
		os.Mkdir(dir, 0600)
	}
}

func (sp *SecurityPrimitives) Init() {
	sp.cleanupDir()

	sp.mkdir(sp.directory)
	sp.mkdir(filepath.Join(sp.directory, "ca"))
	sp.mkdir(filepath.Join(sp.directory, "server"))
	sp.mkdir(filepath.Join(sp.directory, "clients"))
	sp.mkdir(filepath.Join(sp.directory, "common"))
}

func NewOpenVPNSecPrimitives() *SecurityPrimitives {
	dir := "certs"
	return &SecurityPrimitives{
		dir,
		filepath.Join(dir, "ca", "ca.crt"),
		filepath.Join(dir, "ca", "ca.key"),
		filepath.Join(dir, "server", "server.crt"),
		filepath.Join(dir, "server", "server.key"),
		"none",
		filepath.Join("bin", "tls", "crl.pem"),
		filepath.Join(dir, "ta.key"),
		nil,
		nil,
		nil,
		nil,
	}
}

func (sp *SecurityPrimitives) CACert() string {
	return sp.caCert
}

func (sp *SecurityPrimitives) CrlPEM() string {
	return sp.crlPEM
}

func (sp *SecurityPrimitives) DhPEM() string {
	return sp.dhPEM
}

func (sp *SecurityPrimitives) TAKey() string {
	return sp.taKey
}

func (sp *SecurityPrimitives) ServerCert() string {
	return sp.serverCert
}

func (sp *SecurityPrimitives) ServerKey() string {
	return sp.serverKey
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
	//var ca *openssl.CA
	var ca *x509.Certificate

	/*
		ssl := openssl.Openssl{
			Path: "certs", // A storage folder, where to store all certs

			Country:      "GI",
			Province:     "MYST",
			City:         "Blockchain",
			Organization: "Mysterium Network",
			CommonName:   "Mysterium CA",
			Email:        "private@mysterium.network",
		}
	*/

	sp.Init()

	if ca, err = sp.CreateCA(); err != nil {
		log.Info("CreateCA failed: ", err)
		return
	}

	if err = sp.CreateCert(ca, true); err != nil {
		log.Info("CreateCert failed: ", err)
		return
	}

	/*
		if ta, err = ssl.CreateTA(sp.taKey); err != nil {
		log.Println("CreateTA failed: ", err)
			return
		}
		sp.taKey = ta.GetFilePath()
	*/
}
