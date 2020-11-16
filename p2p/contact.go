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

var (
	// ErrContactNotFound represents that no p2p contact is found.
	ErrContactNotFound = errors.New("p2p contact not found")
)

const (
	// ContactTypeNATSv1 is p2p contact type that uses NATS service.
	ContactTypeNATSv1 = "nats/p2p/v1"

	// ContactTypeHTTPv1 is p2p contact type that uses builtin broker.
	ContactTypeHTTPv1 = "broker/p2p/v1"
)

// ContactDefinition represents p2p contact which contains NATS broker addresses for connection.
type ContactDefinition struct {
	Type            string   `json:`
	BrokerAddresses []string `json:"broker_addresses"`
}

// ParseContact tries to parse p2p contact from given contacts list.
func ParseContact(contacts market.ContactList) (list []ContactDefinition, err error) {
	for _, c := range contacts {
		if c.Type == ContactTypeNATSv1 || c.Type == ContactTypeHTTPv1 {
			def, ok := c.Definition.(ContactDefinition)
			if !ok {
				return nil, fmt.Errorf("invalid p2p contact definition: %#v", c.Definition)
			}
			def.Type = c.Type
			list = append(list, def)
		}
	}

	if len(list) > 0 {
		return list, nil
	}

	return nil, ErrContactNotFound
}

// RegisterContactUnserializer registers global proposal contact unserializer.
func RegisterContactUnserializer() {
	market.RegisterContactUnserializer(
		ContactTypeNATSv1,
		func(rawDefinition *json.RawMessage) (market.ContactDefinition, error) {
			var contact ContactDefinition
			err := json.Unmarshal(*rawDefinition, &contact)
			return contact, err
		},
	)
	market.RegisterContactUnserializer(
		ContactTypeHTTPv1,
		func(rawDefinition *json.RawMessage) (market.ContactDefinition, error) {
			var contact ContactDefinition
			err := json.Unmarshal(*rawDefinition, &contact)
			return contact, err
		},
	)
}
