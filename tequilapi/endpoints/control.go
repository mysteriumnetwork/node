/*
 * Copyright (C) 2026 The "MysteriumNetwork/node" Authors.
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

package endpoints

import (
	"io"

	"github.com/gin-gonic/gin"

	"github.com/mysteriumnetwork/go-rest/apierror"
	"github.com/mysteriumnetwork/node/tequilapi/contract"
)

type controlExecutor interface {
	ExecuteJSON(payload []byte) error
}

type controlEndpoint struct {
	executor controlExecutor
}

func (ce *controlEndpoint) Execute(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Error(apierror.ParseFailed())
		return
	}

	if err := ce.executor.ExecuteJSON(payload); err != nil {
		c.Error(apierror.BadRequest(err.Error(), contract.ErrCodeServiceStart))
		return
	}

	c.Status(202)
}

// AddRoutesForControl adds direct control message execution route.
func AddRoutesForControl(executor controlExecutor) func(*gin.Engine) error {
	endpoint := &controlEndpoint{executor: executor}

	return func(e *gin.Engine) error {
		e.POST("/control/messages", endpoint.Execute)
		return nil
	}
}
