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
)

// Primitives structure holds TLS primitives required to setup basic cryptographics for openvpn server/client
type Primitives struct {
	CertificateAuthority *CertificateAuthority
	ServerCertificate    *CertificateKeyPair
	PresharedKey         *TLSPresharedKey
}

// NewTLSPrimitives function creates TLS primitives for given service location and provider id
func NewTLSPrimitives(serviceLocation dto.Location, serviceProviderID identity.Identity) (*Primitives, error) {

	key, err := createTLSCryptKey()
	if err != nil {
		return nil, err
	}

	ca, err := CreateAuthority(newCACert(serviceLocation))
	if err != nil {
		return nil, err
	}

	server, err := ca.CreateDerived(newServerCert(x509.ExtKeyUsageServerAuth, serviceLocation, serviceProviderID))
	if err != nil {
		return nil, err
	}

	return &Primitives{
		CertificateAuthority: ca,
		ServerCertificate:    server,
		PresharedKey:         &key,
	}, nil
}
