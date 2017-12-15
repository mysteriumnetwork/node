package nats_dialog

import (
	"github.com/mysterium/node/communication"
)

type dialog struct {
	communication.Sender
	communication.Receiver
}

func (dialog *dialog) Close() error {
	return nil
}
