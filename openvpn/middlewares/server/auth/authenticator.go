package auth

import (
	"github.com/mysterium/node/server"
)

type Authenticator func(username, password string) (bool, error)

func NewAuthenticator(mysteriumClient server.Client) Authenticator {
	return func(username, password string) (bool, error) {
		return mysteriumClient.AuthenticateClient(
			username,
			password,
		)
	}
}
