package identity

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
)

func SignatureBytes(signatureBytes []byte) Signature {
	return Signature{signatureBytes}
}

func SignatureHex(signature string) Signature {
	signatureBytes, _ := hex.DecodeString(signature)
	return Signature{signatureBytes}
}

type Signature struct {
	raw []byte
}

func (signature *Signature) Hex() string {
	return hex.EncodeToString(signature.raw)
}

func (signature *Signature) Base64() string {
	return base64.StdEncoding.EncodeToString(signature.Bytes())
}

func (signature *Signature) Bytes() []byte {
	return signature.raw
}

func (signature Signature) EqualsTo(other Signature) bool {
	return bytes.Equal(signature.raw, other.raw)
}
