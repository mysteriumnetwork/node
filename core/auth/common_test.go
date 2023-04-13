/*
 * Copyright (C) 2023 The "MysteriumNetwork/node" Authors.
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
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestTokenParsing(t *testing.T) {
	token, err := TokenFromContext(&gin.Context{Request: &http.Request{}})
	assert.NoError(t, err)
	assert.Empty(t, token)

	token, err = TokenFromContext(&gin.Context{Request: &http.Request{Header: map[string][]string{"Authorization": {"Bearer 123"}}}})
	assert.NoError(t, err)
	assert.Equal(t, "123", token)

	token, err = TokenFromContext(&gin.Context{Request: &http.Request{Header: map[string][]string{"Cookie": {"token=321"}}}})
	assert.NoError(t, err)
	assert.Equal(t, "321", token)
}
