package sleep

import "C"
import (
	"github.com/mysteriumnetwork/gowinlog"
	"github.com/rs/zerolog/log"
)

func (n Notifier) Start() {
	log.Debug().Msg("Register for sleep log events")

	watcher, err := winlog.NewWinLogWatcher()
	if err != nil {
		log.Error().Msgf("couldn't create log watcher: %v\n", err)
		return
	}

	watcher.SubscribeFromNow("System", "*[System[Provider[@Name='Microsoft-Windows-Power-Troubleshooter'] and EventID=1]]")
	for {
		select {
		case <-watcher.Event():
			n.eventbus.Publish(AppTopicSleepNotification, EventWakeup)
		case err := <-watcher.Error():
			log.Error().Msgf("Log watcher error: %v\n", err)
		}
	}
	<-n.stop
	watcher.Shutdown()
}

func (n Notifier) Stop() {
	log.Debug().Msg("Unregister sleep log events watcher")
	close(n.stop)
}
