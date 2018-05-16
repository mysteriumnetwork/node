package server

// CredentialsChecker callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsChecker func(username, password string) (bool, error)

// CredentialsCheckerWithClientID callback checks given auth primitives (i.e. customer identity signature / node's sessionId)
type CredentialsCheckerWithClientID func(clientID int, username, password string) (bool, error)
