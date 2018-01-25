package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
)

type sessionLookupF func(session session.SessionID) (session.Session, bool)

type verifierFactory func(identity.Identity) identity.Verifier

type sessionAuthenticator struct {
	sessionLookup  sessionLookupF
	createVerifier verifierFactory
}

//NewSessionAuthenticator provides glue code for openvpn management interface to validate incoming client login request,
//it expects session id as username, and session signature signed by client as password
func NewSessionAuthenticator(sessionLookup sessionLookupF, verifierCreator verifierFactory) *sessionAuthenticator {
	return &sessionAuthenticator{sessionLookup: sessionLookup, createVerifier: verifierCreator}
}

func (sa *sessionAuthenticator) ValidateSession(sessionString, signatureString string) (bool, error) {
	sessionId := session.SessionID(sessionString)
	currentSession, found := sa.sessionLookup(sessionId)
	if !found {
		return false, nil
	}

	verifier := sa.createVerifier(currentSession.PeerIdentity)
	signature := identity.SignatureBase64(signatureString)

	return verifier.Verify([]byte(sessionString), signature), nil
}
