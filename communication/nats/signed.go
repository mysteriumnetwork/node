/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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

package nats

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/mysteriumnetwork/node/identity"
)

var (
	// SignatureEncoding represents base64 variant chosen for signature
	// encoding
	SignatureEncoding = base64.RawURLEncoding
)

// SignedSubject signs topic to pass command through nats-proxy
func SignedSubject(signer identity.Signer, topic string) (string, error) {
	ts := time.Now().Truncate(0).Unix()

	signature, err := signer.Sign([]byte(fmt.Sprintf("%d.%s", ts, topic)))
	if err != nil {
		return "", err
	}

	encodedSignature := SignatureEncoding.EncodeToString(signature.Bytes())

	return fmt.Sprintf("signed.%s.%d.%s", encodedSignature, ts, topic), nil
}
