package auth

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)

// Authenticator returns client's current auth primitives (i.e. customer identity signature / node's sessionId)
type Authenticator func() (username string, password string, err error)

// NewAuthenticatorFake returns Authenticator callback
func NewAuthenticatorFake() Authenticator {
	return func() (username string, password string, err error) {
		username = "valid_user_name"
		password = "valid_password"
		err = nil
		return
	}
}

func NewSignedSessionIdAuthenticator(id session.SessionID, signer identity.Signer) Authenticator {

	signature, err := signer.Sign([]byte(id))

	return func() (string, string, error) {
		return id, signature.Base64(), err
	}
}
