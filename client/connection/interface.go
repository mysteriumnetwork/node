package connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/session"
)

// DialogEstablisherCreator creates new dialog establisher by given identity
type DialogEstablisherCreator func(identity.Identity) communication.DialogEstablisher

// VpnClientCreator creates new vpn client by given session, provider identity and uses state callback to report state changes
type VpnClientCreator func(session.SessionDto, identity.Identity, state.ClientStateCallback) (openvpn.Client, error)

// Manager interface provides methods to manage connection
type Manager interface {
	// Connect creates new connection from given consumer to provider, reports error if connection already exists
	Connect(consumerID identity.Identity, providerID identity.Identity) error
	// Status queries current status of connection
	Status() ConnectionStatus
	// Disconnect closes established connection, reports error if no connection
	Disconnect() error
}
