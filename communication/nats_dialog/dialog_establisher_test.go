package nats_dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats_discovery"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDialogEstablisher_Interface(t *testing.T) {
	var _ communication.DialogEstablisher = &dialogEstablisher{}
}

func TestDialogEstablisher_Factory(t *testing.T) {
	id := identity.FromAddress("123456")
	signer := &identity.SignerFake{}

	establisher := NewDialogEstablisher(id, signer)
	assert.NotNil(t, establisher)
	assert.Equal(t, id, establisher.myIdentity)
	assert.Equal(t, signer, establisher.mySigner)
}

func TestDialogEstablisher_CreateDialog(t *testing.T) {
	connection := nats.StartConnectionFake()
	connection.MockResponse(
		"provider1.dialog-create",
		[]byte(`{
			"payload": {
				"reason": 200,
				"reasonMessage": "OK"
			},
			"signature": "d9183d29a6c95dd604b0c2f29e8799f3ee1c5a36ae1ee66aff274813436e365d69b2ef80573ffc7c76aa746f3509481fd9d3501e37223953da8046fe5fafffb200"
		}`),
	)
	defer connection.Close()

	signer := &identity.SignerFake{}

	establisher := &dialogEstablisher{
		myIdentity: identity.FromAddress("consumer1"),
		mySigner:   signer,
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

	expectedCodec := NewCodecSigner(communication.NewCodecJSON(), signer, identity.NewVerifyIsAuthorized())
	assert.Equal(
		t,
		nats.NewSender(connection, expectedCodec, "provider1.consumer1"),
		dialogNats.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, expectedCodec, "provider1.consumer1"),
		dialogNats.Receiver,
	)
}
