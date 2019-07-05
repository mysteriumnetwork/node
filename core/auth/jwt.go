/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package auth

import (
	"math/rand"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// JwtAuthenticator contains JWT handling methods
type JwtAuthenticator struct {
	storage       Storage
	encryptionKey []byte
}

// JWTToken contains token details
type JWTToken struct {
	Token          string
	ExpirationTime time.Time
}

// JWTCookieName name of the cookie JWT token is stored in
const JWTCookieName string = "token"

type jwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type jwtKey []byte

const expiresIn = 48 * time.Hour
const encryptionKeyBucket = "jwt"
const encryptionKeyName = "jwt-encryption-key"

// NewJWTAuthenticator creates a new JWT authentication instance
func NewJWTAuthenticator(storage Storage) *JwtAuthenticator {
	auth := &JwtAuthenticator{
		storage: storage,
	}

	return auth
}

// CreateToken creates a new JWT token
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

// ValidateToken validates a JWT token
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
