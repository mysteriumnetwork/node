package auth

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)

// CredentialsProvider returns client's current auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsProvider func() (username string, password string, err error)

//NewSignedSessionIdCredentialsProvider returns session id as username and id signed with given signer as password
func NewSignedSessionIdCredentialsProvider(id session.SessionID, signer identity.Signer) CredentialsProvider {
	signature, err := signer.Sign([]byte(id))

	return func() (string, string, error) {
		return string(id), signature.Base64(), err
	}
}
