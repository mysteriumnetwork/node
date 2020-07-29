// +build !darwin

package sleep

import (
	"github.com/mysteriumnetwork/node/eventbus"
	"github.com/rs/zerolog/log"
)

type Notifier struct {
	eventbus eventbus.Publisher
}

func NewNotifier(eventbus eventbus.Publisher) *Notifier {
	return &Notifier{eventbus: eventbus}
}

func (n Notifier) Start() {
	log.Debug().Msg("Register for noop sleep events")
}

func (n Notifier) Stop() {
	log.Debug().Msg("Unregister noop sleep events")
}
