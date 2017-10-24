package identity

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	base64 "encoding/base64"
	"math/big"
)

func GenerateKeys() *ecdsa.PrivateKey {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	return privateKey
}

func Sign(privateKey *ecdsa.PrivateKey, message string) (r *big.Int, s *big.Int, err error) {
	hashed := []byte(message)
	return ecdsa.Sign(rand.Reader, privateKey, hashed)
}

func Verify(publicKey ecdsa.PublicKey, message string, r *big.Int, s *big.Int) bool {
	hashed := []byte(message)
	return ecdsa.Verify(&publicKey, hashed, r, s)
}

func EncodePublicKey(publicKey ecdsa.PublicKey) string {
	x509Encoded, _ := x509.MarshalPKIXPublicKey(&publicKey)
	base64Encoded := base64.StdEncoding.EncodeToString(x509Encoded)
	return base64Encoded
}

func DecodePublicKey(encodedPublicKey string) *ecdsa.PublicKey {
	x509Encoded, _ := base64.StdEncoding.DecodeString(encodedPublicKey)
	genericPublicKey, _ := x509.ParsePKIXPublicKey(x509Encoded)
	publicKey := genericPublicKey.(*ecdsa.PublicKey)
	return publicKey
}

func EncodePrivateKey(privateKey ecdsa.PrivateKey) string {
	x509Encoded, _ := x509.MarshalECPrivateKey(&privateKey)
	base64Encoded := base64.StdEncoding.EncodeToString(x509Encoded)
	return base64Encoded
}

func DecodePrivateKey(encodedPrivateKey string) *ecdsa.PrivateKey {
	x509Encoded, _ := base64.StdEncoding.DecodeString(encodedPrivateKey)
	privateKey, _ := x509.ParseECPrivateKey(x509Encoded)
	return privateKey
}
