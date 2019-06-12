/*
 * Copyright (C) 2019 The "MysteriumNetwork/node" Authors.
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

package auth

import (
	"github.com/msteinert/pam"
)

type pamAuth struct{}

func newAuth() *pamAuth {
	return &pamAuth{}
}

func (pa *pamAuth) Auth(header string) error {
	user, password, err := parseBasicAuthHeader
	if err != nil {
		return err
	}

	t, err := pam.StartFunc("", user, func(_ pam.Style, _ string) (string, error) {
		return password, nil
	})
	if err != nil {
		return err
	}

	return t.Authenticate(0)
}
