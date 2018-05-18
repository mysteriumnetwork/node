package session

import (
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/session"
	"errors"
)

// SignaturePrefix is used to prefix with each session string before calculating signature or extracting identity
const SignaturePrefix = "MystVpnSessionId:"

// Validator structure that keeps attributes needed Validator operations
type Validator struct {
	sessionManager *manager
	identityExtractor identity.Extractor
}

// NewValidator return Validator instance
func NewValidator(m *manager, extractor identity.Extractor) (*Validator) {
	return &Validator{
				sessionManager: m,
				identityExtractor: extractor,
			}
}

// Validate provides glue code for openvpn management interface to validate incoming client login request,
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

// Cleanup removes session from underlying session managers
func (v *Validator) Cleanup(sessionString string) error {
	sessionID := session.SessionID(sessionString)
	_, found := v.sessionManager.FindSession(sessionID)
	if !found {
		return errors.New("no underlying session exists: " + sessionString)
	}

	v.sessionManager.RemoveSession(sessionID)
	return nil
}
