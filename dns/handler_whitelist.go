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
	"fmt"
	"strings"

	"github.com/miekg/dns"
	"github.com/mysteriumnetwork/node/core/policy"
	"github.com/mysteriumnetwork/node/firewall"
	"github.com/rs/zerolog/log"
)

// WhitelistAnswers creates a DNS handler that whitelist resolved queries to firewall.
func WhitelistAnswers(
	resolver dns.Handler,
	trafficBlocker firewall.IncomingTrafficFirewall,
	policies *policy.Repository,
) dns.Handler {
	return &whitelistHandler{
		resolver:       resolver,
		trafficBlocker: trafficBlocker,
		policies:       policies,
	}
}

type whitelistHandler struct {
	resolver       dns.Handler
	trafficBlocker firewall.IncomingTrafficFirewall
	policies       *policy.Repository
}

func (wh *whitelistHandler) ServeDNS(writer dns.ResponseWriter, req *dns.Msg) {
	resolverWriter := &recordingWriter{writer: writer}
	wh.resolver.ServeDNS(resolverWriter, req)
	resp := resolverWriter.responseMsg

	if err := wh.whitelistByAnswer(resp); err != nil {
		log.Warn().Err(err).Msgf("Error updating firewall by DNS query: %s", resp.String())

		resp := &dns.Msg{}
		resp.SetRcode(req, dns.RcodeNameError)
		writer.WriteMsg(resp)
		return
	}

	writer.WriteMsg(resp)
}

func (wh *whitelistHandler) whitelistByAnswer(response *dns.Msg) error {
	for _, record := range response.Answer {
		switch recordValue := record.(type) {
		case *dns.A:
			if err := wh.whitelistByARecord(recordValue); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown record type: %s", dns.Type(record.Header().Rrtype))
		}
	}
	return nil
}

func (wh *whitelistHandler) whitelistByARecord(record *dns.A) error {
	host := strings.TrimRight(record.Hdr.Name, ".")
	ip := record.A

	if wh.policies.IsHostAllowed(host) {
		_, err := wh.trafficBlocker.AllowIPAccess(ip)
		return err
	}

	return nil
}
