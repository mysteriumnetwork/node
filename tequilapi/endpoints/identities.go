package endpoints

import (
	"net/http"

	"encoding/json"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/identity"
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
	idDto := identityDto{string(*id)}
	if err != nil {
		utils.SendError(writer, err, http.StatusInternalServerError) // This should never happen
		return
	}

	utils.WriteAsJson(idDto, writer)
}

func (endpoint *identitiesApi) Register(writer http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	registerReq, err := toRegisterRequest(request)
	if err != nil {
		utils.SendError(writer, err, http.StatusBadRequest)
		return
	}
	errorMap := validateRegistrationRequest(registerReq)
	if errorMap.HasErrors() {
		utils.SendValidationErrorMessage(writer, errorMap)
		return
	}

	// TODO: Register here

}

func toCreateRequest(req *http.Request) (*identityCreationDto, error) {
	var identityCreationReq = &identityCreationDto{}
	err := json.NewDecoder(req.Body).Decode(&identityCreationReq)
	if err != nil {
		return nil, err
	}
	return identityCreationReq, nil
}

func toRegisterRequest(req *http.Request) (*identityRegistrationDto, error) {
	var idRegDto = identityRegistrationDto{}
	err := json.NewDecoder(req.Body).Decode(&idRegDto)
	if err != nil {
		return nil, err
	}
	return &idRegDto, nil
}

func validateRegistrationRequest(regReq *identityRegistrationDto) (errors *validation.FieldErrorMap) {
	errors = validation.NewErrorMap()
	return
}

func validateCreationRequest(createReq *identityCreationDto) (errors *validation.FieldErrorMap) {
	errors = validation.NewErrorMap()
	return
}

func RegisterIdentitiesEndpoint(router *httprouter.Router, idm identity.IdentityManagerInterface) {
	idmEnd := NewIdentitiesEndpoint(idm)
	router.GET("/identities", idmEnd.List)
	router.POST("/identities", idmEnd.Create)
	//router.GET("/identities/:id/registration", idmEnd.IsRegister)
	router.PUT("/identities/:id/registration", idmEnd.Register)
}
