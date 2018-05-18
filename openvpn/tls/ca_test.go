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

package tls

import (
	"crypto/x509"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

var fakeServiceLocation = dto.Location{"GB", "", ""}
var fakeProviderID = identity.Identity{"some fake identity "}

func TestCertificateAuthorityIsCreatedAndCertCanBeSerialized(t *testing.T) {
	_, err := CreateAuthority(newCACert(fakeServiceLocation))
	assert.NoError(t, err)
}

func TestServerCertificateIsCreatedAndBothCertAndKeyCanBeSerialized(t *testing.T) {
	ca, err := CreateAuthority(newCACert(fakeServiceLocation))
	assert.NoError(t, err)
	_, err = ca.CreateDerived(newServerCert(x509.ExtKeyUsageServerAuth, fakeServiceLocation, fakeProviderID))
	assert.NoError(t, err)
}
