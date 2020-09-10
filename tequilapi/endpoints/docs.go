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
	"fmt"
	"net/http"
	"text/template"

	"github.com/julienschmidt/httprouter"
)

const docsSwaggerTemplate = `<!DOCTYPE html>
<html>
<head>
    <title>ReDoc</title>
    <!-- needed for adaptive design -->
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">

    <!--
    ReDoc doesn't change outer page styles
    -->
    <style>
        body {
            margin: 0;
            padding: 0;
        }
    </style>
</head>
<body>
<redoc spec-url="{{.SpecURL}}"></redoc>
<script src="https://cdn.jsdelivr.net/npm/redoc@next/bundles/redoc.standalone.js"> </script>
</body>
</html>
`

type docsSwaggerVars struct {
	SpecURL string
}

// NewDocsEndpoint creates and returns documentation endpoint.
func NewDocsEndpoint() (*DocsEndpoint, error) {
	docsTpl, err := template.New("swagger_index.html").Parse(docsSwaggerTemplate)
	if err != nil {
		return nil, fmt.Errorf("cant parse swagger template: %w", err)
	}

	return &DocsEndpoint{
		docsTpl: docsTpl,
		docsVars: docsSwaggerVars{
			SpecURL: "tequilapi.json",
		},
	}, nil
}

// DocsEndpoint serves API documentation.
type DocsEndpoint struct {
	docsTpl  *template.Template
	docsVars docsSwaggerVars
}

// Index redirects root route to swagger docs.
func (se *DocsEndpoint) Index(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	http.Redirect(resp, request, "/docs/index.html", http.StatusMovedPermanently)
}

// Docs middleware to serve the API docs.
func (se *DocsEndpoint) Docs(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	se.docsTpl.Execute(resp, se.docsVars)
}

// AddRoutesForDocs attaches documentation endpoints to router.
func AddRoutesForDocs(router *httprouter.Router) error {
	endpoint, err := NewDocsEndpoint()
	if err != nil {
		return err
	}

	router.GET("/", endpoint.Index)
	router.GET("/docs", endpoint.Docs)
	return nil
}
