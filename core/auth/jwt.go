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
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// JWTAuthenticator contains JWT handling methods
type JWTAuthenticator struct {
	encryptionKey []byte
}

// JWT contains token details
type JWT struct {
	Token          string
	ExpirationTime time.Time
}

// JWTEncryptionKey contains the encryption key for JWT
type JWTEncryptionKey []byte

// JWTCookieName name of the cookie JWT token is stored in
const JWTCookieName string = "token"

type jwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

const expiresIn = 48 * time.Hour

// NewJWTAuthenticator creates a new JWT authentication instance
func NewJWTAuthenticator(encryptionKey JWTEncryptionKey) *JWTAuthenticator {
	auth := &JWTAuthenticator{
		encryptionKey,
	}

	return auth
}

// CreateToken creates a new JWT token
func (jwtAuth *JWTAuthenticator) CreateToken(username string) (JWT, error) {
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
		return JWT{}, err
	}

	return JWT{Token: tokenString, ExpirationTime: expirationTime}, nil
}

// ValidateToken validates a JWT token
func (jwtAuth *JWTAuthenticator) ValidateToken(token string) (bool, error) {
	claims := &jwtClaims{}

	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtAuth.encryptionKey, nil
	})
	if err != nil {
		return false, err
	}

	if tkn == nil || !tkn.Valid {
		return false, errors.New("invalid JWT token")
	}

	return true, nil
}

func (jwtAuth *JWTAuthenticator) getExpirationTime() time.Time {
	return time.Now().Add(expiresIn)
}
