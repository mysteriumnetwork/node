package bytescount

import (
	"github.com/pkg/errors"
	"time"
)

// NewSelectiveStatsHandler creates and returns composite handler, which invokes internal handler at given interval
func NewSelectiveStatsHandler(handler SessionStatsHandler, clock func() time.Time, interval time.Duration) (SessionStatsHandler, error) {
	if interval < 0 {
		return nil, errors.New("Invalid 'interval' parameter")
	}
	var lastTime *time.Time = nil
	return func(sessionStats SessionStats) error {
		now := clock()
		if lastTime == nil || (now.Sub(*lastTime)) >= interval {
			lastTime = &now
			return handler(sessionStats)
		}
		return nil
	}, nil
}
