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

package p2p

import (
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
)

const (
	// TopicKeepAlive is keep alive endpoint.
	TopicKeepAlive = "p2p-keepalive"

	// TopicSessionCreate is a session create endpoint for p2p communication.
	TopicSessionCreate = "p2p-session-create"
	// TopicSessionAcknowledge is a session acknowledge endpoint for p2p communication.
	TopicSessionAcknowledge = "p2p-session-acknowledge"
	// TopicSessionStatus is a session status notification for p2p communication.
	TopicSessionStatus = "p2p-session-connectivity-status"
	// TopicSessionDestroy is a session destroy endpoint for p2p communication.
	TopicSessionDestroy = "p2p-session-destroy"

	// TopicPaymentMessage is a payment messages endpoint for p2p communication.
	TopicPaymentMessage = "p2p-payment-message"
	// TopicPaymentInvoice is a payment invoices endpoint for p2p communication.
	TopicPaymentInvoice = "p2p-payment-invoice"
)

// Message represent message with data bytes.
type Message struct {
	Data []byte
}

// UnmarshalProto is convenient helper to unmarshal message data into strongly typed proto message.
func (m *Message) UnmarshalProto(to proto.Message) error {
	return proto.Unmarshal(m.Data, to)
}

// ProtoMessage is convenient helper to return message with marshaled proto data bytes.
func ProtoMessage(m proto.Message) *Message {
	pbBytes, err := proto.Marshal(m)
	if err != nil {
		// In general this should never happen as passed input value
		// should implement proto.Message interface.
		log.Err(err).Msg("Failed to marshal proto message")
		return &Message{Data: []byte{}}
	}
	return &Message{Data: pbBytes}
}

const (
	statusCodeOK                 = 1
	statusCodePublicErr          = 2
	statusCodeInternalErr        = 3
	statusCodeHandlerNotFoundErr = 4
)

// transportMsg is internal structure for sending and receiving messages.
type transportMsg struct {
	// Header fields.
	id         uint64
	statusCode uint64
	topic      string
	msg        string

	// Data field.
	data []byte
}

func (m *transportMsg) readFrom(conn wireReader) error {
	return conn.readMsg(m)
}

func (m *transportMsg) writeTo(conn wireWriter) error {
	return conn.writeMsg(m)
}
