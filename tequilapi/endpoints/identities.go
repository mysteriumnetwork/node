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
	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/blockchain/registration"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/server"
	"github.com/mysterium/node/tequilapi/utils"
	"github.com/mysterium/node/tequilapi/validation"
)

// swagger:model IdentityDTO
type identityDto struct {
	// identity in Ethereum address format
	// required: true
	// example: 0x0000000000000000000000000000000000000001
	ID string `json:"id"`
}

// swagger:model IdentityList
type identityList struct {
	Identities []identityDto `json:"identities"`
}

// swagger:model IdentityCreationDTO
type identityCreationDto struct {
	Passphrase *string `json:"passphrase"`
}

// swagger:model IdentityRegistrationDTO
type identityRegistrationDto struct {
	// value true means register, false - unregister which in not implemented yet
	Registered bool `json:"registered"`
}

// swagger:model IdentityUnlockingDTO
type identityUnlockingDto struct {
	Passphrase string `json:"passphrase"`
}

type identitiesAPI struct {
	idm             identity.IdentityManagerInterface
	mysteriumClient server.Client
	signerFactory   identity.SignerFactory
	proofGenerator  registration.ProofGenerator
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
func NewIdentitiesEndpoint(idm identity.IdentityManagerInterface, mystClient server.Client, signerFactory identity.SignerFactory, generator registration.ProofGenerator) *identitiesAPI {
	return &identitiesAPI{idm, mystClient, signerFactory, generator}
}

// swagger:operation GET /identities Identity listIdentities
// ---
// summary: Returns identities
// description: Returns list of identities
// responses:
//   200:
//     description: List of identities
//     schema:
//       "$ref": "#/definitions/IdentityList"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *identitiesAPI) List(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	idArry := endpoint.idm.GetIdentities()
	idsSerializable := identityList{mapIdentities(idArry, idToDto)}

	utils.WriteAsJSON(idsSerializable, resp)
}

// swagger:operation POST /identities Identity createIdentity
// ---
// summary: Creates new identity
// description: Creates identity and stores in keystore encrypted with passphrase
// parameters:
//   - in: body
//     name: body
//     description: Parameter in body (passphrase) required for creating new identity
//     schema:
//       $ref: "#/definitions/IdentityCreationDTO"
// responses:
//   200:
//     description: Identity created
//     schema:
//       "$ref": "#/definitions/IdentityDTO"
//   400:
//     description: Bad Request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   422:
//     description: Parameters validation error
//     schema:
//       "$ref": "#/definitions/ValidationErrorDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
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
// summary: Registers identity
// description: Registers existing identity with Discovery API
// parameters:
// - name: id
//   in: path
//   description: Identity stored in keystore
//   example: "0x0000000000000000000000000000000000000001"
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Parameter in body (registered) required for registering identity
//   schema:
//     $ref: "#/definitions/IdentityRegistrationDTO"
// responses:
//   202:
//     description: Identity registered
//   400:
//     description: Bad request
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   501:
//     description: Not implemented
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
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

// swagger:operation PUT /identities/{id}/unlock Identity unlockIdentity
// ---
// summary: Unlocks identity
// description: Uses passphrase to decrypt identity stored in keystore
// parameters:
// - in: path
//   name: id
//   description: Identity stored in keystore
//   example: "0x0000000000000000000000000000000000000001"
//   type: string
//   required: true
// - in: body
//   name: body
//   description: Parameter in body (passphrase) required for unlocking identity
//   schema:
//     $ref: "#/definitions/IdentityUnlockingDTO"
// responses:
//   202:
//     description: Identity unlocked
//   400:
//     description: Body parsing error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   403:
//     description: Forbidden
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
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

func (endpoint *identitiesAPI) RegistrationStatus(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	proof, err := endpoint.proofGenerator.GenerateProofForIdentity(common.HexToAddress(id))
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	utils.WriteAsJSON(
		struct {
			PublicKeyPart1 *utils.ByteArrayString
			PublicKeyPart2 *utils.ByteArrayString
			Signature_S    *utils.ByteArrayString
			Signature_R    *utils.ByteArrayString
			V              uint8
		}{
			PublicKeyPart1: utils.ToByteArrayString(proof.Data[0:32]),
			PublicKeyPart2: utils.ToByteArrayString(proof.Data[32:64]),
			Signature_S:    utils.ToByteArrayString(proof.Signature.S[:]),
			Signature_R:    utils.ToByteArrayString(proof.Signature.R[:]),
			V:              proof.Signature.V,
		},
		resp,
	)
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
	generator registration.ProofGenerator,
) {
	idmEnd := NewIdentitiesEndpoint(idm, mystClient, signerFactory, generator)
	router.GET("/identities", idmEnd.List)
	router.POST("/identities", idmEnd.Create)
	router.GET("/identities/:id/registration", idmEnd.RegistrationStatus)
	router.PUT("/identities/:id/registration", idmEnd.Register)
	router.PUT("/identities/:id/unlock", idmEnd.Unlock)
}
