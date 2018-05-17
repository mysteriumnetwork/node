package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"github.com/mysterium/node/openvpn/middlewares/client/auth"
)

const sessionSignaturePrefix = "MystVpnSessionId:"


type Validator struct {
	sessionManager *manager
	identityExtractor identity.Extractor
}

func NewValidator(m session.Manager, extractor identity.Extractor) (*Validator) {
	return &Validator{
				sessionManager: NewManager(m),
				identityExtractor: extractor,
			}
}

// NewSessionValidator provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func (v *Validator) Validate(clientID int, sessionString, signatureString string) (bool, error) {
		sessionId := session.SessionID(sessionString)
		currentSession, found := v.sessionManager.FindSession(clientID, sessionId)
		if !found {
			return false, nil
		}

		signature := identity.SignatureBase64(signatureString)
		extractedIdentity, err := v.identityExtractor.Extract([]byte(sessionSignaturePrefix+sessionString), signature)
		if err != nil {
			return false, err
		}
		return currentSession.ConsumerID == extractedIdentity, nil
}

// SignatureCredentialsProvider returns session id as username and id signed with given signer as password
func SignatureCredentialsProvider(id session.SessionID, signer identity.Signer) auth.CredentialsProvider {
	return func() (string, string, error) {
		signature, err := signer.Sign([]byte(sessionSignaturePrefix + id))
		return string(id), signature.Base64(), err
	}
}
