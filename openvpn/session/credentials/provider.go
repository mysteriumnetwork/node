package credentials

import (
	"github.com/mysterium/node/session"
	ovpnsession "github.com/mysterium/node/openvpn/session"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
)

// SignatureCredentialsProvider returns session id as username and id signed with given signer as password
func SignatureCredentialsProvider(id session.SessionID, signer identity.Signer) auth.CredentialsProvider {
	return func() (string, string, error) {
		signature, err := signer.Sign([]byte(ovpnsession.SignaturePrefix + id))
		return string(id), signature.Base64(), err
	}
}
