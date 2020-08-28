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

package service

import (
	"testing"

	"github.com/mysteriumnetwork/go-openvpn/openvpn/management"
	"github.com/mysteriumnetwork/node/core/service"
	"github.com/mysteriumnetwork/node/identity"
	"github.com/mysteriumnetwork/node/pb"
	"github.com/mysteriumnetwork/node/trace"
	"github.com/stretchr/testify/assert"
)

var (
	identityExisting   = identity.FromAddress("deadbeef")
	sessionExisting, _ = service.NewSession(
		&service.Instance{},
		&pb.SessionRequest{Consumer: &pb.ConsumerInfo{Id: identityExisting.Address}},
		trace.NewTracer(""),
	)
	sessionExistingString = string(sessionExisting.ID)
)

func TestValidateReturnsFalseWhenNoSessionFound(t *testing.T) {
	validator := createAuthHandler(identity.Identity{}).validate

	authenticated, err := validator(1, "not important", "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsFalseWhenSignatureIsInvalid(t *testing.T) {
	validator := createAuthHandlerWithSession(identity.FromAddress("wrongsignature"), sessionExisting).validate

	authenticated, err := validator(1, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.False(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValid(t *testing.T) {
	validator := createAuthHandlerWithSession(identityExisting, sessionExisting).validate

	authenticated, err := validator(1, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValidAndClientIDDiffers(t *testing.T) {
	validator := createAuthHandlerWithSession(identityExisting, sessionExisting).validate

	validator(1, sessionExistingString, "not important")
	authenticated, err := validator(2, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestValidateReturnsTrueWhenSessionExistsAndSignatureIsValidAndClientIDMatches(t *testing.T) {
	validator := createAuthHandlerWithSession(identityExisting, sessionExisting).validate

	validator(1, sessionExistingString, "not important")
	authenticated, err := validator(1, sessionExistingString, "not important")

	assert.NoError(t, err)
	assert.True(t, authenticated)
}

func TestSecondClientIsNotDisconnectedWhenFirstClientDisconnects(t *testing.T) {
	var firstClientConnected = []string{
		">CLIENT:CONNECT,1,4",
		">CLIENT:ENV,username=client1",
		">CLIENT:ENV,password=passwd1",
		">CLIENT:ENV,END",
	}

	var secondClientConnected = []string{
		">CLIENT:CONNECT,2,4",
		">CLIENT:ENV,username=client2",
		">CLIENT:ENV,password=passwd2",
		">CLIENT:ENV,END",
	}

	var firstClientDisconnected = []string{
		">CLIENT:DISCONNECT,1,4",
		">CLIENT:ENV,username=client1",
		">CLIENT:ENV,password=passwd1",
		">CLIENT:ENV,END",
	}

	mockMangement := &management.MockConnection{CommandResult: "SUCCESS"}
	middleware := createAuthHandlerWithSession(identityExisting, sessionExisting)
	middleware.Start(mockMangement)

	feedLinesToMiddleware(middleware, firstClientConnected)
	assert.Equal(t, "client-auth-nt 1 4", mockMangement.LastLine)

	feedLinesToMiddleware(middleware, secondClientConnected)
	assert.Equal(t, "client-auth-nt 2 4", mockMangement.LastLine)

	mockMangement.LastLine = ""
	feedLinesToMiddleware(middleware, firstClientDisconnected)
	assert.Empty(t, mockMangement.LastLine)

}

func TestSecondClientWithTheSameCredentialsIsConnected(t *testing.T) {
	var firstClientConnected = []string{
		">CLIENT:CONNECT,1,4",
		">CLIENT:ENV,username=Boop!",
		">CLIENT:ENV,password=V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=",
		">CLIENT:ENV,END",
	}

	var secondClientDisconnected = []string{
		">CLIENT:CONNECT,2,4",
		">CLIENT:ENV,username=Boop!",
		">CLIENT:ENV,password=V6ifmvLuAT+hbtLBX/0xm3C0afywxTIdw1HqLmA4onpwmibHbxVhl50Gr3aRUZMqw1WxkfSIVdhpbCluHGBKsgE=",
		">CLIENT:ENV,END",
	}

	mockMangement := &management.MockConnection{CommandResult: "SUCCESS"}
	middleware := createAuthHandlerWithSession(identityExisting, sessionExisting)
	middleware.Start(mockMangement)

	feedLinesToMiddleware(middleware, firstClientConnected)
	assert.Equal(t, "client-auth-nt 1 4", mockMangement.LastLine)

	feedLinesToMiddleware(middleware, secondClientDisconnected)
	assert.Equal(t,
		"client-auth-nt 2 4",
		mockMangement.LastLine,
		"second authentication with the same credentials but with different clientID should succeed",
	)
}

func feedLinesToMiddleware(middleware management.Middleware, lines []string) {
	for _, line := range lines {
		middleware.ConsumeLine(line)
	}
}
