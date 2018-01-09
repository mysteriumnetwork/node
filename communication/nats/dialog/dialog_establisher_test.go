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
	assert.Equal(t, id, establisher.myId)
	assert.Equal(t, signer, establisher.mySigner)
}

func TestDialogEstablisher_CreateDialog(t *testing.T) {
	myId := identity.FromAddress("0x6B21b441D0D2Fa1d86407977A3a5C6eD90Ff1A62")
	peerId := identity.FromAddress("0x0d1a35e53b7f3478d00B7C23838C0D48b2a81017")

	connection := nats.StartConnectionFake()
	connection.MockResponse(
		"peer-topic.dialog-create",
		[]byte(`{
			"payload": {"reason":200,"reasonMessage":"OK"},
            "signature": "iaV65n3kEve9+EzwWVi65qJFrb4FQZwq4yWdVH++abts3mW/xqKHpPKro7kX/liFRZgV5RHQMjE+TzPPdeJfewA="
		}`),
	)
	defer connection.Close()

	signer := &identity.SignerFake{}
	establisher := mockEstablisher(myId, connection, signer)

	dialogInstance, err := establisher.CreateDialog(peerId, dto_discovery.Contact{})
	defer dialogInstance.Close()
	assert.NoError(t, err)
	assert.NotNil(t, dialogInstance)

	dialogNats, ok := dialogInstance.(*dialog)
	assert.True(t, ok)

	expectedCodec := NewCodecSecured(communication.NewCodecJSON(), signer, identity.NewVerifierIdentity(peerId))
	assert.Equal(
		t,
		nats.NewSender(connection, expectedCodec, "peer-topic."+myId.Address),
		dialogNats.Sender,
	)
	assert.Equal(
		t,
		nats.NewReceiver(connection, expectedCodec, "peer-topic."+myId.Address),
		dialogNats.Receiver,
	)
}

func TestDialogEstablisher_CreateDialogWhenResponseHijacked(t *testing.T) {
	myId := identity.FromAddress("0x6B21b441D0D2Fa1d86407977A3a5C6eD90Ff1A62")
	peerId := identity.FromAddress("0x0d1a35e53b7f3478d00B7C23838C0D48b2a81017")

	connection := nats.StartConnectionFake()
	connection.MockResponse(
		"peer-topic.dialog-create",
		[]byte(`{
			"payload": {"reason":200,"reasonMessage":"OK"},
			"signature": "2Rg9KabJXdYEsMLynoeZ8+4cWjauHuZq/ydIE0NuNl1psu+AVz/8fHaqdG81CUgf2dNQHjciOVPagEb+X6//sgA="
		}`),
	)
	defer connection.Close()

	establisher := mockEstablisher(myId, connection, &identity.SignerFake{})

	dialogInstance, err := establisher.CreateDialog(peerId, dto_discovery.Contact{})
	defer dialogInstance.Close()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dialog creation error. failed to unpack response 'peer-topic.dialog-create'. invalid message signature ")
	assert.Nil(t, dialogInstance)

	_, ok := dialogInstance.(*dialog)
	assert.True(t, ok)
}

func mockEstablisher(myId identity.Identity, connection nats.Connection, signer identity.Signer) *dialogEstablisher {
	peerTopic := "peer-topic"

	return &dialogEstablisher{
		myId:     myId,
		mySigner: signer,
		peerAddressFactory: func(contact dto_discovery.Contact) (*discovery.AddressNATS, error) {
			return discovery.NewAddressWithConnection(connection, peerTopic), nil
		},
	}
}
