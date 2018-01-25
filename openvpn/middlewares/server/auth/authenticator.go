package auth

// AuthenticatorChecker callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type AuthenticatorChecker func(username, password string) (bool, error)

// NewCheckerFake returns AuthenticatorChecker callback
func NewCheckerFake() AuthenticatorChecker {
	// TODO: implement
	return func(username, password string) (bool, error) {
		if username == "bad" {
			return false, nil
		}

		return true, nil
	}
}
