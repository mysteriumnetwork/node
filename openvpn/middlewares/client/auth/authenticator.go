package auth

type Authenticator func() (username string, password string, err error)

func NewAuthenticator() Authenticator {
	return func() (username string, password string, err error) {
		username = "valid_user_name"
		password = "valid_password"
		err = nil
		return
	}
}
