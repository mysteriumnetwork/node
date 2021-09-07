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

	if bytes.Compare(msg.data, msg2.data) != 0 {
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
		&transportMsg{
			id: 1,
			statusCode: 1,
			topic: "topic1",
			msg: "msg1",
			data: []byte("data1"),
		},
		&transportMsg{
			id: 2,
			statusCode: 2,
			topic: "topic2",
			msg: "msg2",
			data: []byte("data2"),
		},
		&transportMsg{
			id: 3,
			statusCode: 3,
			topic: "topic3",
			msg: "msg3",
			data: []byte("data3"),
		},
		&transportMsg{
			id: 4,
			statusCode: 4,
			topic: "topic4",
			msg: "msg4",
			data: []byte("data4"),
		},
		&transportMsg{
			id: 5,
			statusCode: 5,
			topic: "topic5",
			msg: "msg5",
			data: []byte("data5"),
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
