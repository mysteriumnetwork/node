package identity

import (
	"github.com/mysterium/node/service_discovery/dto"
)

type IdentityManagerInterface interface {
	CreateNewIdentity(string) (dto.Identity, error)
	GetIdentities() []dto.Identity
	GetIdentity(string) (dto.Identity, error)
	HasIdentity(string) bool
	Register(dto.Identity) error
}
