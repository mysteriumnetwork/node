package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/tequilapi/utils"
)

type identityDto struct {
	Id string `json:"id"`
}

type identitiesApi struct {
	idm identity.IdentityManagerInterface
}

func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface) *identitiesApi {
	return &identitiesApi{idm}
}

func (endpoint *identitiesApi) List(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := endpoint.idm.GetIdentities()

	idsSerializable := make([]identityDto, len(idArry))
	for i, id := range idArry {
		idsSerializable[i] = identityDto{
			Id: string(id),
		}
	}

	utils.WriteAsJson(idsSerializable, writer)
}
