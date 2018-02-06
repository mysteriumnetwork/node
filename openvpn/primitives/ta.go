package primitives

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	log "github.com/cihub/seelog"
	"os"
)

type SymmetricKey []byte

func (sp *SecurityPrimitives) CreateTA(filename string) error {
	var taKey SymmetricKey
	taKey = make([]byte, 256)
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

	var keyEntries []string
	keyEntries = append(keyEntries, "-----BEGIN OpenVPN Static key V1-----\n")
	keyEntries = append(keyEntries, hex.EncodeToString(taKey))
	keyEntries = append(keyEntries, "\n-----END OpenVPN Static key V1-----\n")

	for _, s := range keyEntries {
		_, err := keyOut.WriteString(s)
		if err != nil {
			return err
		}
	}

	log.Debug("written " + sp.taKeyPath)

	return nil
}
