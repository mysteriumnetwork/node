package nats

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestClientInterface(t *testing.T) {
	var _ communication.Client = &clientNats{}
}

func TestNewClient(t *testing.T) {
	identity := dto_discovery.Identity("123456")
	client := NewClient(identity)

	assert.NotNil(t, client)
	assert.Equal(t, identity, client.myIdentity)
}

func TestClientCreateDialog(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()

	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	requestSent := make(chan bool)
	_, err := connection.Subscribe("server1.dialog-create", func(message *nats.Msg) {
		assert.JSONEq(t, `{"identity_id":"client1"}`, string(message.Data))
		connection.Publish(message.Reply, []byte(`{"accepted":true}`))
		requestSent <- true
	})
	assert.Nil(t, err)

	address := nats_discovery.NewAddressForIdentity(dto_discovery.Identity("server1"))
	client := &clientNats{
		myIdentity: dto_discovery.Identity("client1"),
	}
	sender, receiver, err := client.CreateDialog(address.GetContact())
	assert.NoError(t, err)
	assert.NotNil(t, sender)
	assert.NotNil(t, receiver)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}
