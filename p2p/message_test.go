/*
 * Copyright (C) 2021 The "MysteriumNetwork/node" Authors.
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
	"bytes"
	"crypto/rand"
	"testing"
)

// Size of Ethereum signature, for instance
const randomDataSize = 65
const fuzzingIterations = 1_000

func randomData() ([]byte, error) {
	data := make([]byte, randomDataSize)
	_, err := rand.Read(data)
	return data, err
}

func roundTrip(t *testing.T) {
	var out bytes.Buffer
	conn := newProtobufWireWriter(&out)
	conn2 := newProtobufWireReader(&out)

	data, err := randomData()
	if err != nil {
		t.Fatalf("Can't generate random data portion: %v", err)
	}

	msg := transportMsg{
		topic: "test",
		data:  data,
	}
	err = msg.writeTo(conn)
	if err != nil {
		t.Fatalf("Can't write data into conn: %v", err)
	}

	var msg2 transportMsg
	err = msg2.readFrom(conn2)
	if err != nil {
		t.Fatalf("Can't read data from conn2: %v", err)
	}

	if !bytes.Equal(msg.data, msg2.data) {
		t.Fatalf("Data wasn't properly recovered: original %x != recovered %x", msg.data, msg2.data)
	}
}

func TestTransportMessageFuzzing(t *testing.T) {
	for i := 0; i < fuzzingIterations; i++ {
		roundTrip(t)
	}
}

func TestTransportMessagePiping(t *testing.T) {
	var out bytes.Buffer
	conn := newProtobufWireWriter(&out)
	conn2 := newProtobufWireReader(&out)

	msgs := []*transportMsg{
		{
			id:         1,
			statusCode: 1,
			topic:      "topic1",
			msg:        "msg1",
			data:       []byte("data1"),
		},
		{
			id:         2,
			statusCode: 2,
			topic:      "topic2",
			msg:        "msg2",
			data:       []byte("data2"),
		},
		{
			id:         3,
			statusCode: 3,
			topic:      "topic3",
			msg:        "msg3",
			data:       []byte("data3"),
		},
		{
			id:         4,
			statusCode: 4,
			topic:      "topic4",
			msg:        "msg4",
			data:       []byte("data4"),
		},
		{
			id:         5,
			statusCode: 5,
			topic:      "topic5",
			msg:        "msg5",
			data:       []byte("data5"),
		},
	}

	for _, msg := range msgs {
		err := msg.writeTo(conn)
		if err != nil {
			t.Fatalf("msg.writeTo() error: %v", err)
		}
	}

	for _, refMsg := range msgs {
		msg := new(transportMsg)
		err := msg.readFrom(conn2)
		if err != nil {
			t.Fatalf("msg.readFrom() error: %v", err)
		}

		if !(msg.id == refMsg.id &&
			msg.statusCode == refMsg.statusCode &&
			msg.topic == refMsg.topic &&
			msg.msg == refMsg.msg &&
			bytes.Compare(msg.data, refMsg.data) == 0) {
			t.Fatal("channel transport messages are not equal!")
		}
	}

}
