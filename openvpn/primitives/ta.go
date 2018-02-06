package primitives

import (
	"crypto/rand"
	"encoding/pem"
	"fmt"
	log "github.com/cihub/seelog"
	"os"
)

func (sp *SecurityPrimitives) CreateTA(filename string) error {
	taKey := make([]byte, 2048)
	_, err := rand.Read(taKey)
	if err != nil {
		fmt.Println("error:", err)
		return err
	}

	keyOut, err := os.OpenFile(sp.taKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Info("failed to open "+sp.taKeyPath+" for writing:", err)
		return err
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, pemBlockForKey(taKey)); err != nil {
		log.Info("failed to PEM encode TLS auth key", err)
	}

	log.Debug("written " + sp.taKeyPath + "\n")

	return nil
}
