/*
 * Copyright (C) 2020 The "MysteriumNetwork/node" Authors.
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

package p2p

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mysteriumnetwork/node/market"
)

// ErrContactNotFound represents that no p2p contact is found.
var ErrContactNotFound = errors.New("p2p contact not found")

const (
	// ContactTypeV1 is p2p contact type.
	ContactTypeV1 = "nats/p2p/v1"
)

// ContactDefinition represents p2p contact which contains NATS broker addresses for connection.
type ContactDefinition struct {
	BrokerAddresses []string `json:"broker_addresses"`
}

// ParseContact tries to parse p2p contact from given contacts list.
func ParseContact(contacts market.ContactList) (ContactDefinition, error) {
	for _, c := range contacts {
		if c.Type == ContactTypeV1 {
			def, ok := c.Definition.(ContactDefinition)
			if !ok {
				return ContactDefinition{}, fmt.Errorf("invalid p2p contact definition: %#v", c.Definition)
			}
			return def, nil
		}
	}
	return ContactDefinition{}, ErrContactNotFound
}

// RegisterContactUnserializer registers global proposal contact unserializer.
func RegisterContactUnserializer() {
	market.RegisterContactUnserializer(
		ContactTypeV1,
		func(rawDefinition *json.RawMessage) (market.ContactDefinition, error) {
			var contact ContactDefinition
			err := json.Unmarshal(*rawDefinition, &contact)
			return contact, err
		},
	)
}
