package bytescount

import (
	"errors"
	"time"
)

// NewIntervalStatsHandler creates and returns composite handler, which invokes internal handler at given interval
func NewIntervalStatsHandler(handler SessionStatsHandler, currentTime func() time.Time, interval time.Duration) (SessionStatsHandler, error) {
	if interval < 0 {
		return nil, errors.New("Invalid 'interval' parameter")
	}

	firstTime := true
	var lastTime time.Time
	return func(sessionStats SessionStats) error {
		now := currentTime()
		if firstTime || (now.Sub(lastTime)) >= interval {
			firstTime = false
			lastTime = now
			return handler(sessionStats)
		}
		return nil
	}, nil
}
