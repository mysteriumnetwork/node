package discovery

import "time"

// Broker Constants
const (
	BrokerPort             = 4222
	BrokerMaxReconnect     = -1
	BrokerReconnectWait    = 4 * time.Second
	BrokerTimeout          = 5 * time.Second
)
