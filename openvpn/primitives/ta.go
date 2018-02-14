package primitives

import (
	"crypto/rand"
	"encoding/hex"
	log "github.com/cihub/seelog"
	"os"
)

// createTLSCryptKey generates symmetric key in HEX format 2048 bits length
func (p *SecurityPrimitives) createTLSCryptKey() error {

	if err := p.cleanup(p.TLSCryptKeyPath); err != nil {
		return err
	}

	taKey := make([]byte, 256)
	_, err := rand.Read(taKey)
	if err != nil {
		log.Error(logPrefix, "failed to create random key:", err)
		return err
	}

	keyOut, err := os.OpenFile(p.TLSCryptKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Error(logPrefix, "failed to open "+p.TLSCryptKeyPath+" for writing:", err)
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

	log.Debug(logPrefix, "written "+p.TLSCryptKeyPath)

	return nil
}
