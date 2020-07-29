package sleep

import "C"

//export NotifySleep
func NotifySleep() {
	eventChannel <- EventSleep
}

//export NotifyWake
func NotifyWake() {
	eventChannel <- EventWakeup
}
