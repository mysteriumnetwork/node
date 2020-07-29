package sleep

import "github.com/mysteriumnetwork/node/eventbus"

type Event int

const (
	// AppTopicSleepNotification represents sleep management Event notification
	AppTopicSleepNotification = "sleep_notification"

	EventWakeup Event = iota
	EventSleep
)

var eventChannel chan Event

type Notifier struct {
	eventbus eventbus.Publisher
	stop     chan struct{}
}

func NewNotifier(eventbus eventbus.Publisher) *Notifier {
	eventChannel = make(chan Event)
	return &Notifier{
		eventbus: eventbus,
		stop:     make(chan struct{}),
	}
}
