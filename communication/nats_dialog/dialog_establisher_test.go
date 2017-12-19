package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
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
	connection := nats.StartConnectionFake()
	connection.MockResponse(
		"provider1.dialog-create",
		[]byte(`{
			"reason":200,
			"reasonMessage": "OK"
		}`),
	)
	defer connection.Close()

	establisher := &dialogEstablisher{
		myIdentity: dto_discovery.Identity("consumer1"),
		contactAddressFactory: func(contact dto_discovery.Contact) (*nats_discovery.NatsAddress, error) {
			assert.Exactly(t, dto_discovery.Contact{}, contact)
			return nats_discovery.NewAddressWithConnection(connection, "provider1"), nil
		},
	}
	dialogInstance, err := establisher.CreateDialog(dto_discovery.Contact{})
	defer dialogInstance.Close()
	assert.NoError(t, err)
	assert.NotNil(t, dialogInstance)

	dialogNats, ok := dialogInstance.(*dialog)
	assert.True(t, ok)
	assert.Equal(
		t,
		nats.NewSender(connection, "provider1.consumer1"),
		dialogNats.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, "provider1.consumer1"),
		dialogNats.Receiver,
	)
}
