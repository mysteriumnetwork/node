package connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/state"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/session"
)

// DialogCreator creates new dialog between consumer and provider, using given contact information
type DialogCreator func(consumerID, providerID identity.Identity, contact dto.Contact) (communication.Dialog, error)

// VpnClientCreator creates new vpn client by given session,
// consumer identity, provider identity and uses state callback to report state changes
type VpnClientCreator func(session.SessionDto, identity.Identity, identity.Identity, state.Callback) (openvpn.Client, error)

// Manager interface provides methods to manage connection
type Manager interface {
	// Connect creates new connection from given consumer to provider, reports error if connection already exists
	Connect(consumerID identity.Identity, providerID identity.Identity) error
	// Status queries current status of connection
	Status() ConnectionStatus
	// Disconnect closes established connection, reports error if no connection
	Disconnect() error
}
