/*
 * Copyright (C) 2018 The Mysterium Network Authors
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

type Primitives struct {
	CertificateAuthority *CertificateAuthority
	ServerCertificate    *CertificateKeyPair
	PresharedKey         *TLSPresharedKey
}

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
