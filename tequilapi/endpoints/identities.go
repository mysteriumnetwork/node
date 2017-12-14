package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
)

type identityDto struct {
	Id dto.Identity `json:"id"`
}
type identitiesListDto struct {
	Identities []identityDto `json:"identitiesListDto"`
}

type identitiesApi struct {
	idm identity.IdentityManagerInterface
}

func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface) *identitiesApi {
	return &identitiesApi{idm}
}

func (endpoint *identitiesApi) List(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := endpoint.idm.GetIdentities()

	idsSerializable := identitiesListDto{
		Identities: make([]identityDto, len(idArry)),
	}

	for k, v := range idArry {
		idsSerializable.Identities[k] = identityDto{v}
	}

	utils.WriteAsJson(idsSerializable, writer)
}
