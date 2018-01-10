package endpoints

import (
	"net/http"

	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/tequilapi/validation"
)

type identityDto struct {
	Id string `json:"id"`
}

type identityList struct {
	Identities []identityDto `json:"identities"`
}

type identityCreationDto struct {
	Password string `json:"password"`
}

type identityRegistrationDto struct {
	Registered bool `json:"registered"`
}

type identitiesApi struct {
	idm             identity.IdentityManagerInterface
	mysteriumClient server.Client
	signerFactory   identity.SignerFactory
}

func idToDto(id identity.Identity) identityDto {
	return identityDto{id.Address}
}

func mapIdentities(idArry []identity.Identity, f func(identity.Identity) identityDto) (idDtoArry []identityDto) {
	idDtoArry = make([]identityDto, len(idArry))
	for i, id := range idArry {
		idDtoArry[i] = f(id)
	}
	return
}

//NewIdentitiesEndpoint creates identities api controller used by tequilapi service
func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface, mystClient server.Client, signerFactory identity.SignerFactory) *identitiesApi {
	return &identitiesApi{idm, mystClient, signerFactory}
}

func (endpoint *identitiesApi) List(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := endpoint.idm.GetIdentities()
	idsSerializable := identityList{mapIdentities(idArry, idToDto)}

	utils.WriteAsJson(idsSerializable, resp)
}

func (endpoint *identitiesApi) Create(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	createReq, err := toCreateRequest(request)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}
	errorMap := validateCreationRequest(createReq)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(resp, errorMap)
		return
	}
	id, err := endpoint.idm.CreateNewIdentity(createReq.Password)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	idDto := idToDto(id)
	utils.WriteAsJson(idDto, resp)
}

func (endpoint *identitiesApi) Register(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := identity.FromAddress(params.ByName("id"))
	registerReq, err := toRegisterRequest(request)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}
	err = validateRegistrationRequest(registerReq)
	if err != nil {
		utils.SendError(resp, err, http.StatusNotImplemented)
		return
	}

	err = endpoint.mysteriumClient.RegisterIdentity(id, endpoint.signerFactory(id))
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
}

func toCreateRequest(req *http.Request) (*identityCreationDto, error) {
	var identityCreationReq = &identityCreationDto{}
	err := json.NewDecoder(req.Body).Decode(&identityCreationReq)
	if err != nil {
		return nil, err
	}
	return identityCreationReq, nil
}

func toRegisterRequest(req *http.Request) (isRegisterReq identityRegistrationDto, err error) {
	isRegisterReq = identityRegistrationDto{}
	err = json.NewDecoder(req.Body).Decode(&isRegisterReq)
	return
}

func validateRegistrationRequest(regReq identityRegistrationDto) (err error) {
	if regReq.Registered == false {
		err = errors.New("Unregister not supported")
	}
	return
}

func validateCreationRequest(createReq *identityCreationDto) (errors *validation.FieldErrorMap) {
	errors = validation.NewErrorMap()
	if len(createReq.Password) == 0 {
		errors.ForField("password").AddError("required", "Field is required")
	}
	return
}

//AddRoutesForIdentities creates /identities endpoint on tequilapi service
func AddRoutesForIdentities(
	router *httprouter.Router,
	idm identity.IdentityManagerInterface,
	mystClient server.Client,
	signerFactory identity.SignerFactory,
) {
	idmEnd := NewIdentitiesEndpoint(idm, mystClient, signerFactory)
	router.GET("/identities", idmEnd.List)
	router.POST("/identities", idmEnd.Create)
	router.PUT("/identities/:id/registration", idmEnd.Register)
}
