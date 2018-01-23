package auth

// Authenticator callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type Authenticator func(username, password string) (bool, error)

// NewAuthenticatorFake returns Authenticator callback
func NewAuthenticatorFake() Authenticator {
	// TODO: implement
	return func(username, password string) (bool, error) {
		if username == "bad" {
			return false, nil
		}

		return true, nil
	}
}
