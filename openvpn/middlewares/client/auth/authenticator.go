package auth

// Authenticator returns client's current auth primitives (i.e. customer identity signature / node's sessionId)
type Authenticator func() (username string, password string, err error)

// NewAuthenticatorFake returns Authenticator callback
func NewAuthenticatorFake() Authenticator {
	return func() (username string, password string, err error) {
		username = "valid_user_name"
		password = "valid_password"
		err = nil
		return
	}
}
