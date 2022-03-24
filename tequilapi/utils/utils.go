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

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/rs/zerolog/log"
)

// WriteAsJSON writes a given value `v` to a given http.ResponseWritter
// forcing `content-type application/json`. Optional httpCode parameter
// can be given to also write a specific status code.
func WriteAsJSON(v interface{}, writer http.ResponseWriter, httpCode ...int) {
	writer.Header().Set("Content-type", "application/json; charset=utf-8")

	blob, err := json.Marshal(v)
	if err != nil {
		http.Error(writer, "Http response write error", http.StatusInternalServerError)
		return
	}

	if len(httpCode) > 0 {
		writer.WriteHeader(httpCode[0])
	}

	if _, err := writer.Write(blob); err != nil {
		log.Error().Err(err).Msg("Writing response body failed")
	}
}

// swagger:model APIError
type apiErrorSwagger struct {
	apierror.APIError
}

// ForwardError writes err to the response if it's in `apierror.APIError` format.
// Otherwise, appends it to the fallback APIError's message.
func ForwardError(c *gin.Context, err error, fallback *apierror.APIError) {
	var apiErr *apierror.APIError
	if errors.As(err, &apiErr) {
		c.Error(err)
	} else {
		fallback.Err.Message = fallback.Err.Message + ": " + fmt.Errorf("%w", err).Error()
		c.Error(fallback)
	}
}
