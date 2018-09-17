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
	"crypto/x509/pkix"
	"testing"

	"github.com/stretchr/testify/assert"
)

var caSubject = pkix.Name{
	Country:            []string{"GB"},
	Organization:       []string{""},
	OrganizationalUnit: []string{""},
}

var serverCertSubject = pkix.Name{
	Country:            []string{"GB"},
	Organization:       []string{""},
	OrganizationalUnit: []string{""},
	CommonName:         "some fake identity ",
}

func TestCertificateAuthorityIsCreatedAndCertCanBeSerialized(t *testing.T) {
	_, err := CreateAuthority(newCACert(caSubject))
	assert.NoError(t, err)
}

func TestServerCertificateIsCreatedAndBothCertAndKeyCanBeSerialized(t *testing.T) {
	ca, err := CreateAuthority(newCACert(caSubject))
	assert.NoError(t, err)
	_, err = ca.CreateDerived(newServerCert(x509.ExtKeyUsageServerAuth, serverCertSubject))
	assert.NoError(t, err)
}
