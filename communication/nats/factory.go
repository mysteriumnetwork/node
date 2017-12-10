package nats

import (
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

func NewServer(identity dto_discovery.Identity) *serverNats {
	return &serverNats{
		myAddress: nats_discovery.NewAddressForIdentity(identity),
	}
}

func NewClient(identity dto_discovery.Identity) *clientNats {
	return &clientNats{
		myIdentity: identity,
	}
}
