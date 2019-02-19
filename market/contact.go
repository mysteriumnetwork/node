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

package market

import "encoding/json"

// ContactList is list of Contact structures, to have custom JSON marshaling
type ContactList []Contact

// MarshalJSON encodes in such manner that `null` situation avoided in JSON
func (list ContactList) MarshalJSON() ([]byte, error) {
	if list == nil {
		return json.Marshal([]Contact{})
	}
	return json.Marshal([]Contact(list))
}

// Contact represents contact object with type and concrete definition according to type
type Contact struct {
	Type       string            `json:"type"`
	Definition ContactDefinition `json:"definition"`
}

// ContactDefinition is interface for contacts of all types
type ContactDefinition interface {
}

// UnsupportedContactType is a contact which is returned by unserializer when encountering unregistered types of contact
type UnsupportedContactType struct {
}

var _ ContactDefinition = UnsupportedContactType{}

// ContactDefinitionUnserializer represents function which called for concrete contact type to unserialize
type ContactDefinitionUnserializer func(*json.RawMessage) (ContactDefinition, error)

// contact unserializer registry
// TODO avoid global map variables and wrap this functionality into some kind of component?
var contactDefinitionMap = make(map[string]ContactDefinitionUnserializer)

// RegisterContactUnserializer registers unserializer for specified payment method
func RegisterContactUnserializer(paymentMethod string, unserializer func(*json.RawMessage) (ContactDefinition, error)) {
	contactDefinitionMap[paymentMethod] = unserializer
}

func unserializeContacts(message *json.RawMessage) (contactList ContactList) {
	contactList = ContactList{}
	if message == nil {
		return
	}

	// get an array of raw definitions
	var contacts []struct {
		Type       string           `json:"type"`
		Definition *json.RawMessage `json:"definition"`
	}
	if err := json.Unmarshal([]byte(*message), &contacts); err != nil {
		return
	}

	for _, contactItem := range contacts {
		contactList = append(contactList, Contact{
			Type:       contactItem.Type,
			Definition: unserializeContact(contactItem.Type, contactItem.Definition),
		})
	}

	return
}

func unserializeContact(contactType string, rawMessage *json.RawMessage) ContactDefinition {
	fn, ok := contactDefinitionMap[contactType]
	if !ok {
		return UnsupportedContactType{}
	}
	definition, err := fn(rawMessage)
	if err != nil {
		return UnsupportedContactType{}
	}

	return definition
}
