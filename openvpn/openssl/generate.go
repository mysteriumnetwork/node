package openssl

import (
	"fmt"
	"github.com/stamp/go-openssl"
	"log"
	"os"
	"path/filepath"
)

type SecurityPrimitives struct {
	directory  string
	caCert     string
	caKey      string
	serverCert string
	serverKey  string
	dhPEM      string
	crlPEM     string
	taKey      string
}

func NewOpenVPNSecPrimitives() *SecurityPrimitives {
	dir := ""
	return &SecurityPrimitives{
		dir,
		filepath.Join(dir, "ca.crt"),
		filepath.Join(dir, "ca.key"),
		filepath.Join(dir, "server.crt"),
		filepath.Join(dir, "server.key"),
		filepath.Join(dir, "dh.pem"),
		filepath.Join(dir, "crl.pem"),
		filepath.Join(dir, "ta.key"),
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

func cleanupDir(dir string) {
	err := RemoveContents(dir)
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
	var ca *openssl.CA
	var cert *openssl.Cert
	var dh *openssl.DH
	var ta *openssl.TA

	ssl := openssl.Openssl{
		Path: "certs", // A storage folder, where to store all certs

		Country:      "GI",
		Province:     "MYST",
		City:         "Blockchain",
		Organization: "Mysterium Network",
		CommonName:   "Mysterium CA",
		Email:        "private@mysterium.network",
	}

	cleanupDir(ssl.Path)

	if ca, err = ssl.CreateCA(sp.caCert, sp.caKey); err != nil {
		log.Println("CreateCA failed: ", err)
		return
	}
	sp.caCert = ca.GetFilePath()
	sp.crlPEM = ca.GetCRLPath()

	if cert, err = ssl.CreateCert(sp.serverCert, sp.serverKey, "server", ca, true); err != nil {
		log.Println("CreateCert failed: ", err)
		return
	}
	sp.serverCert = cert.GetFilePath()
	sp.serverKey = cert.GetKeyPath()

	if dh, err = ssl.CreateDH(sp.dhPEM, 1024); err != nil {
		log.Println("CreateDH failed: ", err)
		return
	}
	sp.dhPEM = dh.GetFilePath()

	if ta, err = ssl.CreateTA(sp.taKey); err != nil {
		log.Println("CreateTLSCryptKey failed: ", err)
		return
	}
	sp.taKey = ta.GetFilePath()
}
