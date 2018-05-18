package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"errors"
)

const SignaturePrefix = "MystVpnSessionId:"

type Validator struct {
	sessionManager *Manager
	identityExtractor identity.Extractor
}

func NewValidator(m *Manager, extractor identity.Extractor) (*Validator) {
	return &Validator{
				sessionManager: m,
				identityExtractor: extractor,
			}
}

// NewSessionValidator provides glue code for openvpn management interface to validate incoming client login request,
// it expects session id as username, and session signature signed by client as password
func (v *Validator) Validate(clientID int, sessionString, signatureString string) (bool, error) {
		sessionID := session.SessionID(sessionString)
		currentSession, found := v.sessionManager.FindUpdateSession(clientID, sessionID)
		if !found {
			return false, nil
		}

		signature := identity.SignatureBase64(signatureString)
		extractedIdentity, err := v.identityExtractor.Extract([]byte(SignaturePrefix+sessionString), signature)
		if err != nil {
			return false, err
		}
		return currentSession.ConsumerID == extractedIdentity, nil
}

// Removes session from underlying session managers
func (v *Validator) Cleanup(sessionString string) error {
	sessionID := session.SessionID(sessionString)
	_, found := v.sessionManager.FindSession(sessionID)
	if !found {
		return errors.New("no underlying session exists: " + sessionString)
	}

	v.sessionManager.RemoveSession(sessionID)
	return nil
}
