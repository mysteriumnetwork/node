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

// Signature structure
type Signature struct {
	raw []byte
}

// Base64Encode encodes signature into Base64 format
func (signature *Signature) Base64Encode() string {
	return base64.StdEncoding.EncodeToString(signature.Bytes())
}

// SignatureBase64Decode decodes base64 string signature into raw bytes format
func SignatureBase64Decode(signature string) Signature {
	signatureBytes, _ := base64.StdEncoding.DecodeString(signature)
	return Signature{signatureBytes}
}

// Bytes returns signature in raw bytes format
func (signature *Signature) Bytes() []byte {
	return signature.raw
}

// EqualsTo compares current signature with a given one
func (signature Signature) EqualsTo(other Signature) bool {
	return bytes.Equal(signature.raw, other.raw)
}
