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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
)

// CertificateKeyPair represents x509 type certificate and corresponding private key
type CertificateKeyPair struct {
	privateKey *ecdsa.PrivateKey
	x509cert   *x509.Certificate
	certBytes  []byte
	keyBytes   []byte
}

// ToPEMFormat method returns certificate serialized to string by PEM encoding rules
func (ckp *CertificateKeyPair) ToPEMFormat() string {
	return string(pem.EncodeToMemory(pemBlock("CERTIFICATE", ckp.certBytes)))
}

// KeyToPEMFormat returns private key serialized to string by PEM encoding rules
func (ckp *CertificateKeyPair) KeyToPEMFormat() string {
	return string(pem.EncodeToMemory(pemBlock("EC PRIVATE KEY", ckp.keyBytes)))
}

func pemBlock(blockType string, data []byte) *pem.Block {
	return &pem.Block{
		Type:  blockType,
		Bytes: data,
	}
}

// CertificateAuthority represents self-signed certificate/key pair which can create signed derived certificates
type CertificateAuthority struct {
	CertificateKeyPair
}

// CreateDerived creates new certificate/key by given x509 data and signed by current authority
func (ca *CertificateAuthority) CreateDerived(template *x509.Certificate) (*CertificateKeyPair, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, ca.x509cert, &privateKey.PublicKey, ca.privateKey)
	if err != nil {
		return nil, err
	}
	return &CertificateKeyPair{
		privateKey: privateKey,
		x509cert:   template,
		certBytes:  certBytes,
		keyBytes:   keyBytes,
	}, nil
}

// CreateAuthority creates new self signed certificate with given x509 data
func CreateAuthority(template *x509.Certificate) (*CertificateAuthority, error) {

	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	keyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, err
	}

	return &CertificateAuthority{
		CertificateKeyPair{
			privateKey: privateKey,
			x509cert:   template,
			certBytes:  certBytes,
			keyBytes:   keyBytes,
		},
	}, nil
}
