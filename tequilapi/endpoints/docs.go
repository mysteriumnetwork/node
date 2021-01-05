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

	"github.com/julienschmidt/httprouter"
	"github.com/mysteriumnetwork/node/tequilapi/endpoints/assets"
)

// NewDocsEndpoint creates and returns documentation endpoint.
func NewDocsEndpoint() *DocsEndpoint {
	return &DocsEndpoint{}
}

// DocsEndpoint serves API documentation.
type DocsEndpoint struct{}

// Index redirects root route to swagger docs.
func (se *DocsEndpoint) Index(resp http.ResponseWriter, request *http.Request, _ httprouter.Params) {
	http.Redirect(resp, request, "/docs/", http.StatusMovedPermanently)
}

// AddRoutesForDocs attaches documentation endpoints to router.
func AddRoutesForDocs(router *httprouter.Router) {
	endpoint := NewDocsEndpoint()
	router.GET("/", endpoint.Index)
	router.ServeFiles("/docs/*filepath", assets.DocsAssets)
}
