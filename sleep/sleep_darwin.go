package sleep

// #cgo LDFLAGS: -framework CoreFoundation -framework IOKit
// void NotifyWake();
// void NotifySleep();
// #include "darwin.h"
import "C"
import (
	"github.com/rs/zerolog/log"
)

func (n Notifier) Start() {
	log.Debug().Msg("Register for sleep events")
	go C.registerNotifications()
	go func() {
		for {
			e := <-eventChannel
			n.eventbus.Publish(AppTopicSleepNotification, e)
		}
	}()
	<-n.stop
}

func (n Notifier) Stop() {
	log.Debug().Msg("Unregister sleep events")
	C.unregisterNotifications()
	close(n.stop)
}
