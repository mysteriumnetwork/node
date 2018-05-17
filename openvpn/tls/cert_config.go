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
	"crypto/x509/pkix"
	"github.com/mysterium/node/identity"
	"github.com/mysterium/node/service_discovery/dto"
	"math/big"
	"time"
)

func newCACert(serviceLocation dto.Location) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(1111),
		Subject: pkix.Name{
			Country:            []string{serviceLocation.Country},
			Organization:       []string{"Mystermium.network"},
			OrganizationalUnit: []string{"Mysterium Team"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		SubjectKeyId:          []byte{1, 2, 3, 4, 5},
		BasicConstraintsValid: true,
		IsCA:        true,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
}
func newServerCert(extUsage x509.ExtKeyUsage, serviceLocation dto.Location, providerID identity.Identity) *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(2222),
		Subject: pkix.Name{
			Country:            []string{serviceLocation.Country},
			CommonName:         providerID.Address,
			Organization:       []string{"Mysterium node operator company"},
			OrganizationalUnit: []string{"Node operator team"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{extUsage},
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
}
