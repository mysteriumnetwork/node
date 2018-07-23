package registry

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

type SignatureDTO struct {
	R string
	S string
	V uint8
}

type PublicKeyPartsDTO struct {
	Part1 string
	Part2 string
}

type RegistrationDataDTO struct {
	Registered bool
	PublicKey  *PublicKeyPartsDTO `json:"PublicKey,omitempty"`
	Signature  *SignatureDTO      `json:"Signature,omitempty"`
}

type registrationEndpoint struct {
	dataProvider   RegistrationDataProvider
	statusProvider RegistrationStatusProvider
}

func newRegistrationEndpoint(dataProvider RegistrationDataProvider, statusProvider RegistrationStatusProvider) *registrationEndpoint {
	return &registrationEndpoint{
		dataProvider:   dataProvider,
		statusProvider: statusProvider,
	}
}

func (endpoint *registrationEndpoint) RegistrationData(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	identity := common.HexToAddress(id)

	isRegistered, err := endpoint.statusProvider.IsRegistered(identity)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	registrationResponse := RegistrationDataDTO{
		Registered: isRegistered,
	}

	if isRegistered {
		utils.WriteAsJSON(registrationResponse, resp)
		return
	}

	registrationData, err := endpoint.dataProvider.ProvideRegistrationData(identity)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	registrationResponse.PublicKey = &PublicKeyPartsDTO{
		Part1: common.ToHex(registrationData.Data[0:32]),
		Part2: common.ToHex(registrationData.Data[32:64]),
	}

	registrationResponse.Signature = &SignatureDTO{
		R: common.ToHex(registrationData.Signature.R[:]),
		S: common.ToHex(registrationData.Signature.S[:]),
		V: registrationData.Signature.V,
	}

	utils.WriteAsJSON(registrationResponse, resp)
}

func AddRegistrationEndpoint(router *httprouter.Router, dataProvider RegistrationDataProvider, statusProvider RegistrationStatusProvider) {

	registrationEndpoint := newRegistrationEndpoint(
		dataProvider,
		statusProvider,
	)

	router.GET("/identities/:id/registration", registrationEndpoint.RegistrationData)

}
