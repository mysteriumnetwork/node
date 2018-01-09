package dialog

import (
	"fmt"
	log "github.com/cihub/seelog"
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/communication/nats"
	"github.com/mysterium/node/communication/nats/discovery"
	"github.com/mysterium/node/identity"
	dto_discovery "github.com/mysterium/node/service_discovery/dto"
)

// NewDialogEstablisher constructs new DialogEstablisher which works thru NATS connection.
func NewDialogEstablisher(myId identity.Identity, signer identity.Signer) *dialogEstablisher {

	return &dialogEstablisher{
		myId:     myId,
		mySigner: signer,
		peerAddressFactory: func(contact dto_discovery.Contact) (*discovery.AddressNATS, error) {
			address, err := discovery.NewAddressForContact(contact)
			if err == nil {
				err = address.Connect()
			}

			return address, err
		},
	}
}

const establisherLogPrefix = "[NATS.DialogEstablisher] "

type dialogEstablisher struct {
	myId               identity.Identity
	mySigner           identity.Signer
	peerAddressFactory func(contact dto_discovery.Contact) (*discovery.AddressNATS, error)
}

func (establisher *dialogEstablisher) CreateDialog(
	peerId identity.Identity,
	peerContact dto_discovery.Contact,
) (communication.Dialog, error) {
	var dialog *dialog

	log.Info(establisherLogPrefix, fmt.Sprintf("Connecting to: %#v", peerContact))
	peerAddress, err := establisher.peerAddressFactory(peerContact)
	if err != nil {
		return dialog, fmt.Errorf("failed to connect to: %#v. %s", peerContact, err)
	}

	peerCodec := establisher.newCodecToPeer(peerId)

	peerSender := establisher.newSenderToPeer(peerAddress, peerCodec)
	err = establisher.negotiateDialog(peerSender)
	if err != nil {
		return dialog, err
	}

	dialog = establisher.newDialogToPeer(peerAddress, peerCodec)
	log.Info(establisherLogPrefix, fmt.Sprintf("Dialog established with: %#v", peerContact))

	return dialog, nil
}

func (establisher *dialogEstablisher) negotiateDialog(sender communication.Sender) error {
	response, err := sender.Request(&dialogCreateProducer{
		&dialogCreateRequest{
			IdentityId: establisher.myId.Address,
		},
	})
	if err != nil {
		return fmt.Errorf("dialog creation error. %s", err)
	}
	if response.(*dialogCreateResponse).Reason != 200 {
		return fmt.Errorf("dialog creation rejected. %#v", response)
	}

	return nil
}

func (establisher *dialogEstablisher) newCodecToPeer(peerId identity.Identity) *codecSecured {

	return NewCodecSecured(
		communication.NewCodecJSON(),
		establisher.mySigner,
		identity.NewVerifierIdentity(peerId),
	)
}

func (establisher *dialogEstablisher) newSenderToPeer(
	peerAddress *discovery.AddressNATS,
	peerCodec *codecSecured,
) communication.Sender {

	return nats.NewSender(
		peerAddress.GetConnection(),
		peerCodec,
		peerAddress.GetTopic(),
	)
}

func (establisher *dialogEstablisher) newDialogToPeer(
	peerAddress *discovery.AddressNATS,
	peerCodec *codecSecured,
) *dialog {

	subTopic := peerAddress.GetTopic() + "." + establisher.myId.Address
	return &dialog{
		Sender:   nats.NewSender(peerAddress.GetConnection(), peerCodec, subTopic),
		Receiver: nats.NewReceiver(peerAddress.GetConnection(), peerCodec, subTopic),
	}
}
