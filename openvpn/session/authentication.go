package session

import (
	"github.com/mysterium/node/identity"
	client_auth "github.com/mysterium/node/openvpn/middlewares/client/auth"
	server_auth "github.com/mysterium/node/openvpn/middlewares/server/auth"
	"github.com/mysterium/node/session"
)

const sessionSignaturePrefix = "MystVpnSessionId:"

// SignatureCredentialsProvider returns session id as username and id signed with given signer as password
func SignatureCredentialsProvider(id session.SessionID, signer identity.Signer) client_auth.CredentialsProvider {
	return func() (string, string, error) {
		signature, err := signer.Sign([]byte(sessionSignaturePrefix + id))
		return string(id), signature.Base64(), err
	}
}

type sessionFinder func(session session.SessionID) (session.Session, bool)

// NewSessionValidator provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func NewSessionValidator(sessionLookup sessionFinder, extractor identity.Extractor) server_auth.CredentialsChecker {
	return func(sessionString, signatureString string) (bool, error) {
		sessionId := session.SessionID(sessionString)
		currentSession, found := sessionLookup(sessionId)
		if !found {
			return false, nil
		}

		signature := identity.SignatureBase64(signatureString)
		extractedIdentity, err := extractor.Extract([]byte(sessionSignaturePrefix+sessionString), signature)
		if err != nil {
			return false, err
		}
		return currentSession.ConsumerID == extractedIdentity, nil
	}
}
