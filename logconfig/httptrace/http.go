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

	"github.com/mysteriumnetwork/node/logconfig"
	"github.com/rs/zerolog/log"
)

// TraceRequestResponse logs HTTP request and response if log.http is enabled.
func TraceRequestResponse(req *http.Request, resp *http.Response) {
	if !logconfig.CurrentLogOptions.LogHTTP {
		return
	}
	dumpRequest, _ := httputil.DumpRequest(req, true)
	dumpResponse, _ := httputil.DumpResponse(resp, true)
	log.Debug().Msgf("\n== Request == \n%v\n== Response ==\n%v\n", string(dumpRequest), string(dumpResponse))
}
