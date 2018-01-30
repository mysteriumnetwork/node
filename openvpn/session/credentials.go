package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
	"github.com/mysterium/node/session"
)

//NewSignedSessionIdCredentialsProvider returns session id as username and id signed with given signer as password
func NewSignedSessionIdCredentialsProvider(id session.SessionID, signer identity.Signer) auth.CredentialsProvider {
	signature, err := signer.Sign([]byte(id))

	return func() (string, string, error) {
		return string(id), signature.Base64(), err
	}
}
