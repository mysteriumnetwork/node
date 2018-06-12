/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

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

// swagger:model
type identityDto struct {
	ID string `json:"id"`
}

// swagger:model
type identityList struct {
	Identities []identityDto `json:"identities"`
}

type identityCreationDto struct {
	Passphrase *string `json:"passphrase"`
}

type identityRegistrationDto struct {
	Registered bool `json:"registered"`
}

type identityUnlockingDto struct {
	Passphrase string `json:"passphrase"`
}

type identitiesAPI struct {
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
func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface, mystClient server.Client, signerFactory identity.SignerFactory) *identitiesAPI {
	return &identitiesAPI{idm, mystClient, signerFactory}
}

func (endpoint *identitiesAPI) List(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := endpoint.idm.GetIdentities()
	idsSerializable := identityList{mapIdentities(idArry, idToDto)}

	utils.WriteAsJSON(idsSerializable, resp)
}

// swagger:operation POST /identities Identity createIdentity
// ---
// summary: Create new identity
// description: Inner description
// responses:
//   "200":
//     "description": "Success"
//     "schema":
//       "$ref": "#/definitions/identityDto"
//   "400":
//     "$ref": "#/responses/badRequest"
//   "422":
//     "$ref": "#/definitions/validationErrorMessage"
//   "500":
//     "$ref": "#/responses/internalServerError"
func (endpoint *identitiesAPI) Create(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
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
	id, err := endpoint.idm.CreateNewIdentity(*createReq.Passphrase)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	idDto := idToDto(id)
	utils.WriteAsJSON(idDto, resp)
}

// swagger:operation PUT /identities/{id}/registration Identity registerIdentity
// ---
// summary: Registers provided identity
// description: Inner description
// parameters:
// - name: id
//   in: path
//   description: Ethereum Address, example: 0x0000000000000000000000000000000000000001
//   type: string
//   required: true
// responses:
//   "202":
//     "description": "Accepted. Empty body"
//   "400":
//     "$ref": "#/responses/badRequest"
//   "500":
//     "$ref": "#/responses/internalServerError"
//   "501":
//     "$ref": "#/responses/notImplemented"
func (endpoint *identitiesAPI) Register(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
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

func (endpoint *identitiesAPI) Unlock(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")
	unlockReq, err := toUnlockRequest(request)
	if err != nil {
		utils.SendError(resp, err, http.StatusBadRequest)
		return
	}

	err = endpoint.idm.Unlock(id, unlockReq.Passphrase)
	if err != nil {
		utils.SendError(resp, err, http.StatusForbidden)
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

func toUnlockRequest(req *http.Request) (isUnlockingReq identityUnlockingDto, err error) {
	isUnlockingReq = identityUnlockingDto{}
	err = json.NewDecoder(req.Body).Decode(&isUnlockingReq)
	return
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
	if createReq.Passphrase == nil {
		errors.ForField("passphrase").AddError("required", "Field is required")
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
	router.PUT("/identities/:id/unlock", idmEnd.Unlock)
}
