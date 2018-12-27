package session

import "time"

// TimeTracker tracks elapsed time from the beginning of the session
// it's passive (no internal go routines) and simply encapsulates time operation: now - startOfSession expressed as duration
type TimeTracker struct {
	started time.Time
	getTime func() time.Time
}

// NewTracker initializes TimeTracker with specified monotonically increasing clock function (usually time.Now is enough - but we do DI for test sake)
func NewTracker(getTime func() time.Time) TimeTracker {
	return TimeTracker{
		getTime: getTime,
	}
}

func (tt *TimeTracker) StartTracking() {
	tt.started = tt.getTime()
}

func (tt TimeTracker) Elapsed() time.Duration {
	return tt.getTime().Sub(tt.started)
}
