// +build !darwin

package sleep

import (
	"github.com/rs/zerolog/log"
)

func (n Notifier) Start() {
	log.Debug().Msg("Register for noop sleep events")
}

func (n Notifier) Stop() {
	log.Debug().Msg("Unregister noop sleep events")
}
