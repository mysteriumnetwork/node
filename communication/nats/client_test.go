package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClientInterface(t *testing.T) {
	var _ communication.Client = &clientNats{}
}

func TestClientCreateDialog(t *testing.T) {
	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	requestSent := make(chan bool)
	_, err := connection.Subscribe("server1-topic.dialog-create", func(message *nats.Msg) {
		assert.JSONEq(t, `{"identity_id":"client1"}`, string(message.Data))
		connection.Publish(message.Reply, []byte(`{"accepted":true}`))
		requestSent <- true
	})
	assert.Nil(t, err)

	client := &clientNats{
		myIdentity: dto.Identity("client1"),
		connection: connection,
	}
	sender, receiver, err := client.CreateDialog(ContactNATSV1{
		Topic: "server1-topic",
	})
	assert.NoError(t, err)
	assert.NotNil(t, sender)
	assert.NotNil(t, receiver)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}
