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

package dns

import (
	"net"

	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// ResolveViaSystem creates proxying DNS handler.
func ResolveViaSystem() (dns.Handler, error) {
	handler := &proxyHandler{
		client: &dns.Client{},
	}
	if err := handler.configure(); err != nil {
		return nil, errors.Wrap(err, "failed to find system DNS configuration")
	}

	return handler, nil
}

type proxyHandler struct {
	proxyAddrs []string
	client     *dns.Client
}

// configure configures proxy to use system DNS servers.
func (ph *proxyHandler) configure() (err error) {
	cfg, err := configuration()
	if err != nil {
		return err
	}
	for _, server := range cfg.Servers {
		ph.proxyAddrs = append(ph.proxyAddrs, net.JoinHostPort(server, cfg.Port))
	}
	return nil
}

func (ph *proxyHandler) ServeDNS(writer dns.ResponseWriter, req *dns.Msg) {
	for _, addr := range ph.proxyAddrs {
		resp, _, err := ph.client.Exchange(req, addr)
		if err != nil {
			log.Error().Err(err).Msg("Error proxying DNS query to " + addr)
			continue
		}

		writer.WriteMsg(resp)
		return
	}

	resp := &dns.Msg{}
	resp.SetRcode(req, dns.RcodeServerFailure)
	writer.WriteMsg(resp)
}
