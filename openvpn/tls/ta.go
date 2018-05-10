package tls

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

type TLSPresharedKey []byte

func (key TLSPresharedKey) ToPEMFormat() string {
	buffer := bytes.Buffer{}

	fmt.Fprintln(&buffer, "-----BEGIN OpenVPN Static key V1-----")
	fmt.Fprintln(&buffer, hex.EncodeToString(key))
	fmt.Fprintln(&buffer, "-----END OpenVPN Static key V1-----")

	return buffer.String()
}

// createTLSCryptKey generates symmetric key in HEX format 2048 bits length
func createTLSCryptKey() (TLSPresharedKey, error) {

	taKey := make([]byte, 256)
	_, err := rand.Read(taKey)
	if err != nil {
		return nil, err
	}
	return TLSPresharedKey(taKey), nil
}
