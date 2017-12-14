package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/nats-io/go-nats"
	"github.com/nats-io/go-nats/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDialogEstablisher_Interface(t *testing.T) {
	var _ communication.DialogEstablisher = &dialogEstablisher{}
}

func TestDialogEstablisher_Factory(t *testing.T) {
	identity := dto_discovery.Identity("123456")
	establisher := NewDialogEstablisher(identity)

	assert.NotNil(t, establisher)
	assert.Equal(t, identity, establisher.myIdentity)
}

func TestDialogEstablisher_CreateDialog(t *testing.T) {
	server := test.RunDefaultServer()
	defer server.Shutdown()

	connection := test.NewDefaultConnection(t)
	defer connection.Close()

	requestSent := make(chan bool)
	_, err := connection.Subscribe("provider1.dialog-create", func(message *nats.Msg) {
		assert.JSONEq(t, `{"identity_id":"consumer1"}`, string(message.Data))
		connection.Publish(message.Reply, []byte(`{"accepted":true}`))
		requestSent <- true
	})
	assert.Nil(t, err)

	contactAddress := nats_discovery.NewAddressForIdentity(dto_discovery.Identity("provider1"))
	establisher := &dialogEstablisher{
		myIdentity: dto_discovery.Identity("consumer1"),
	}
	dialog, err := establisher.CreateDialog(contactAddress.GetContact())
	assert.NoError(t, err)
	assert.NotNil(t, dialog)

	if err := test.Wait(requestSent); err != nil {
		t.Fatal("Request not sent")
	}
}
