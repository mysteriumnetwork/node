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

package httptrace

import (
	"net/http"
	"net/http/httputil"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/mysteriumnetwork/node/logconfig"
)

var log = logconfig.NewLogger()

// HTTPTraceLog adapter for go-retryablehttp
type HTTPTraceLog struct {
}

// Printf printf
func (*HTTPTraceLog) Printf(format string, params ...interface{}) {
	log.Tracef(format, params...)
}

// LogRequest log request
func (*HTTPTraceLog) LogRequest(logger retryablehttp.Logger, r *http.Request, retryNumber int) {
	if !log.IsTrace() {
		return
	}
	dumpRequest, _ := httputil.DumpRequest(r, true)
	log.Tracef("Request: %v", string(dumpRequest))
}

// LogResponse log response
func (*HTTPTraceLog) LogResponse(logger retryablehttp.Logger, response *http.Response) {
	if !log.IsTrace() {
		return
	}
	dumpResponse, _ := httputil.DumpResponse(response, true)
	log.Tracef("Response: %v", string(dumpResponse))
}
