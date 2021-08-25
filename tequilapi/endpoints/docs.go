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

package endpoints

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mysteriumnetwork/node/tequilapi/endpoints/assets"
)

// NewDocsEndpoint creates and returns documentation endpoint.
func NewDocsEndpoint() *DocsEndpoint {
	return &DocsEndpoint{}
}

// DocsEndpoint serves API documentation.
type DocsEndpoint struct{}

// Index redirects root route to swagger docs.
func (se *DocsEndpoint) Index(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/docs")
}

// AddRoutesForDocs attaches documentation endpoints to router.
func AddRoutesForDocs(c *gin.Engine) error {
	endpoint := NewDocsEndpoint()
	c.GET("/", endpoint.Index)
	c.StaticFS("/docs", assets.DocsAssets)
	return nil
}
