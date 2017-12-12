package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
)

type IdentityData struct {
	Id dto.Identity `json:"id"`
}
type Identities struct {
	Identities []IdentityData `json:"identities"`
}

type IdentitiesApi struct {
	idm identity.IdentityManagerInterface
}

func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface) *IdentitiesApi {
	return &IdentitiesApi{idm}
}

func (endpoint *IdentitiesApi) List(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := endpoint.idm.GetIdentities()

	idsSerializable := Identities{
		Identities: make([]IdentityData, len(idArry)),
	}

	for k, v := range idArry {
		idsSerializable.Identities[k] = IdentityData{v}
	}

	utils.WriteAsJson(idsSerializable, writer)
}
