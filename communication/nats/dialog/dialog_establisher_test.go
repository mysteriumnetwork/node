package dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats/discovery"
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
            "signature": "2Rg9KabJXdYEsMLynoeZ8+4cWjauHuZq/ydIE0NuNl1psu+AVz/8fHaqdG81CUgf2dNQHjciOVPagEb+X6//sgA="
		}`))
	defer connection.Close()

	signer := &identity.SignerFake{}

	establisher := &dialogEstablisher{
		myIdentity: identity.FromAddress("consumer1"),
		mySigner:   signer,
		contactAddressFactory: func(contact dto_discovery.Contact) (*discovery.AddressNATS, error) {
			assert.Exactly(t, dto_discovery.Contact{}, contact)
			return discovery.NewAddressWithConnection(connection, "provider1"), nil
		},
	}
	dialogInstance, err := establisher.CreateDialog(dto_discovery.Contact{})
	defer dialogInstance.Close()
	assert.NoError(t, err)
	assert.NotNil(t, dialogInstance)

	dialogNats, ok := dialogInstance.(*dialog)
	assert.True(t, ok)

	expectedCodec := NewCodecSecured(communication.NewCodecJSON(), signer, identity.NewVerifierSigned())
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
