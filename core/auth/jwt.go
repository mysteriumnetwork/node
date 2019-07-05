package auth

import (
	"math/rand"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

type JwtAuthenticator struct {
	storage       Storage
	encryptionKey []byte
}

type JWTToken struct {
	Token          string
	ExpirationTime time.Time
}

const JWTCookieName string = "token"

type jwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type jwtKey []byte

const expiresIn = 48 * time.Hour
const encryptionKeyBucket = "jwt"
const encryptionKeyName = "jwt-encryption-key"

func NewJWTAuthenticator(storage Storage) *JwtAuthenticator {
	auth := &JwtAuthenticator{
		storage: storage,
	}

	return auth
}

func (jwtAuth *JwtAuthenticator) CreateToken(username string) (JWTToken, error) {
	if err := jwtAuth.ensureEncryptionKeyIsSet(); err != nil {
		return JWTToken{}, err
	}

	expirationTime := jwtAuth.getExpirationTime()
	claims := &jwtClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtAuth.encryptionKey)
	if err != nil {
		return JWTToken{}, err
	}

	return JWTToken{Token: tokenString, ExpirationTime: expirationTime}, nil
}

func (jwtAuth *JwtAuthenticator) ValidateToken(token string) (bool, error) {
	if err := jwtAuth.ensureEncryptionKeyIsSet(); err != nil {
		return false, err
	}
	claims := &jwtClaims{}

	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtAuth.encryptionKey, nil
	})

	if tkn == nil || !tkn.Valid {
		return false, errors.New("Invalid JWT token")
	}
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return false, err
		}
	}

	return true, nil
}

func (jwtAuth *JwtAuthenticator) ensureEncryptionKeyIsSet() error {
	if jwtAuth.encryptionKey == nil {
		return jwtAuth.setupEncryptionKey()
	}

	return nil
}

func (jwtAuth *JwtAuthenticator) setupEncryptionKey() error {
	key := jwtKey{}
	err := jwtAuth.storage.GetValue(encryptionKeyBucket, encryptionKeyName, &key)
	if err != nil {
		key = generateRandomKey(64)
		err := jwtAuth.storage.SetValue(encryptionKeyBucket, encryptionKeyName, key)
		if err != nil {
			return errors.Wrap(err, "Failed to store JWT encryption key")
		}
	}

	jwtAuth.encryptionKey = key

	return nil
}

func (jwtAuth *JwtAuthenticator) getExpirationTime() time.Time {
	return time.Now().Add(expiresIn)
}

func generateRandomKey(length int) []byte {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	key := make([]byte, length)
	for i := range key {
		key[i] = chars[rand.Intn(len(chars))]
	}
	return key
}
