package identity

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
)

// SignatureBytes constructs Signature structure instance from bytes
func SignatureBytes(signatureBytes []byte) Signature {
	return Signature{signatureBytes}
}

// SignatureHex returns Signature struct from hex string
func SignatureHex(signature string) Signature {
	signatureBytes, _ := hex.DecodeString(signature)
	return Signature{signatureBytes}
}

// SignatureBase64 decodes base64 string signature into Signature
func SignatureBase64(signature string) Signature {
	signatureBytes, _ := base64.StdEncoding.DecodeString(signature)
	return Signature{signatureBytes}
}

// Signature structure
type Signature struct {
	raw []byte
}

// Base64 encodes signature into Base64 format
func (signature *Signature) Base64() string {
	return base64.StdEncoding.EncodeToString(signature.Bytes())
}

// Bytes returns signature in raw bytes format
func (signature *Signature) Bytes() []byte {
	return signature.raw
}

// EqualsTo compares current signature with a given one
func (signature Signature) EqualsTo(other Signature) bool {
	return bytes.Equal(signature.raw, other.raw)
}
