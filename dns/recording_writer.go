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
)

type recordingWriter struct {
	writer      dns.ResponseWriter
	responseMsg *dns.Msg
	response    []byte
}

func (rw *recordingWriter) LocalAddr() net.Addr {
	return rw.writer.LocalAddr()
}

func (rw *recordingWriter) RemoteAddr() net.Addr {
	return rw.writer.RemoteAddr()
}

func (rw *recordingWriter) WriteMsg(m *dns.Msg) error {
	rw.responseMsg = m
	return nil
}

func (rw *recordingWriter) Write(m []byte) (int, error) {
	rw.response = m
	return len(m), nil
}

func (rw *recordingWriter) Close() error {
	return rw.writer.Close()
}

func (rw *recordingWriter) TsigStatus() error {
	return rw.writer.TsigStatus()
}

func (rw *recordingWriter) TsigTimersOnly(b bool) {
	rw.writer.TsigTimersOnly(b)
}

func (rw *recordingWriter) Hijack() {
	rw.writer.Hijack()
}
