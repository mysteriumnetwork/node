package dialog

import (
	"github.com/mysterium/node/communication"
	"github.com/mysterium/node/identity"
)

type dialog struct {
	communication.Sender
	communication.Receiver
	peerID identity.Identity
}

func (dialog *dialog) Close() error {
	return nil
}

func (dialog *dialog) GetPeerID() identity.Identity {
	return dialog.peerID
}
