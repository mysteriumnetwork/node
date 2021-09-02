package p2p

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"net/textproto"
	"testing"
)

// Size of Ethereum signature, for instance
const randomDataSize = 65
const fuzzingIterations = 1_000_000

func randomData() ([]byte, error) {
	data := make([]byte, randomDataSize)
	_, err := rand.Read(data)
	return data, err
}

func roundTrip(t *testing.T) {
	var out bytes.Buffer
	conn := textproto.NewWriter(bufio.NewWriter(&out))
	conn2 := textproto.NewReader(bufio.NewReader(&out))

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
