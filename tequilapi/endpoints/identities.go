package endpoints

import (
	"net/http"

	"encoding/json"
	"errors"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/tequilapi/validation"
)

type identityDto struct {
	Id string `json:"id"`
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
}

func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface, mystClient server.Client) *identitiesApi {
	return &identitiesApi{idm, mystClient}
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

func (endpoint *identitiesApi) Create(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	createReq, err := toCreateRequest(request)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}
	errorMap := validateCreationRequest(createReq)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(writer, errorMap)
		return
	}
	id, err := endpoint.idm.CreateNewIdentity(createReq.Password)
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	idDto := identityDto{string(id)}
	utils.WriteAsJson(idDto, writer)
}

func (endpoint *identitiesApi) Register(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := dto.Identity(params.ByName("id"))
	registerReq, err := toRegisterRequest(request)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}
	err = validateRegistrationRequest(registerReq)
	if err != nil {
		utils.SendError(writer, err, 501)
		return
	}

	err = endpoint.mysteriumClient.RegisterIdentity(id)
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError)
		return
	}

	writer.WriteHeader(http.StatusAccepted)
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

func RegisterIdentitiesEndpoint(
	router *httprouter.Router,
	idm identity.IdentityManagerInterface,
	mystClient server.Client,
) {
	idmEnd := NewIdentitiesEndpoint(idm, mystClient)
	router.GET("/identities", idmEnd.List)
	router.POST("/identities", idmEnd.Create)
	router.PUT("/identities/:id/registration", idmEnd.Register)
}
