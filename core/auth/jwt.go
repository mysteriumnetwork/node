package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

type JWTAuthenticator struct {
	encryptionKey []byte
}

type JWT struct {
	Token          string
	ExpirationTime time.Time
}

type JWTEncryptionKey []byte

const JWTCookieName string = "token"

type jwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

const expiresIn = 48 * time.Hour

func NewJWTAuthenticator(encryptionKey JWTEncryptionKey) *JWTAuthenticator {
	auth := &JWTAuthenticator{
		encryptionKey,
	}
	return auth
}

func (jwtAuth *JWTAuthenticator) CreateToken(username string) (JWT, error) {
	expirationTime := jwtAuth.getExpirationTime()
	claims := &jwtClaims{
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtAuth.encryptionKey)
	if err != nil {
		return JWT{}, err
	}

	return JWT{Token: tokenString, ExpirationTime: expirationTime}, nil
}

func (jwtAuth *JWTAuthenticator) ValidateToken(token string) (bool, error) {
	claims := &jwtClaims{}

	tkn, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtAuth.encryptionKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
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
