package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
)

type IdentitiesApi struct {
	getIdentityArray func() []dto.Identity
}

func IdentityHandlers(idm *identity.IdentityManager) *IdentitiesApi {
	return &IdentitiesApi{getIdentityArray: idm.GetIdentities}
}

func (lsid *IdentitiesApi) Get(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := lsid.getIdentityArray()

	idSerializable := dto.Identities{
		Identities: make([]dto.IdentityData, len(idArry)),
	}

	for k, v := range idArry {
		idSerializable.Identities[k] = dto.IdentityData{v}
	}

	utils.WriteAsJson(idSerializable, writer)
}
