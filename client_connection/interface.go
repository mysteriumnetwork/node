package client_connection

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/openvpn"
	"github.com/mysterium/node/openvpn/middlewares/client/state"
	"github.com/mysterium/node/session"
)

type DialogEstablisherCreator func(identity identity.Identity) communication.DialogEstablisher

type VpnClientCreator func(session.SessionDto, identity.Identity, state.ClientStateCallback) (openvpn.Client, error)

type Manager interface {
	Connect(consumerID identity.Identity, providerID identity.Identity) error
	Status() ConnectionStatus
	Disconnect() error
}
