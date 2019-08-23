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

package dns

import (
	"net"

	"github.com/miekg/dns"
	"github.com/mysteriumnetwork/node/logconfig"
)

var log = logconfig.NewLogger()

// ResolveViaConfigured create new dns.Server handler which handles incoming DNS requests
func ResolveViaConfigured() dns.Handler {
	client := &dns.Client{}

	return dns.HandlerFunc(func(writer dns.ResponseWriter, req *dns.Msg) {
		resp := &dns.Msg{}
		resp.SetRcode(req, dns.RcodeServerFailure)

		config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err != nil {
			log.Error("Error loading DNS config: ", err)
			writer.WriteMsg(resp)
			return
		}

		for _, server := range config.Servers {
			forwardAddress := net.JoinHostPort(server, config.Port)
			resp, _, err = client.Exchange(req, forwardAddress)
			if err != nil {
				log.Errorf(`Error proxying DNS query to "%s". %s`, forwardAddress, err)
				continue
			}
		}
		writer.WriteMsg(resp)
	})
}
