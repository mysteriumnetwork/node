package auth

type Authenticator func(username, password string) (bool, error)

func NewAuthenticator() Authenticator {
	return func(username, password string) (bool, error) {
		if username == "bad" {
			return false, nil
		}

		return true, nil
	}
}
