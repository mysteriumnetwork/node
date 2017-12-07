package endpoints

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
)

type identityList struct {
	Identities []dto.Identity `json:"identities"`
}

type listIdentitiesEndpoint struct {
	listIdentities func() []dto.Identity
}

func ListIdentitiesEndpointFactory(idm *identity.IdentityManager) *listIdentitiesEndpoint {
	return &listIdentitiesEndpoint{listIdentities: idm.GetIdentities}
}

func (lsid *listIdentitiesEndpoint) ListIdentities(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	list := identityList{
		Identities: lsid.listIdentities(),
	}

	utils.WriteAsJson(list, writer)
}
