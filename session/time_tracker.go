package session

import "time"

// TimeTracker tracks elapsed time from the beginning of the session
// it's passive (no internal go routines) and simply encapsulates time operation: now - startOfSession expressed as duration
type TimeTracker struct {
	started   bool
	startTime time.Time
	getTime   func() time.Time
}

// NewTracker initializes TimeTracker with specified monotonically increasing clock function (usually time.Now is enough - but we do DI for test sake)
func NewTracker(getTime func() time.Time) TimeTracker {
	return TimeTracker{
		getTime: getTime,
	}
}

func (tt *TimeTracker) StartTracking() {
	tt.started = true
	tt.startTime = tt.getTime()
}

func (tt TimeTracker) Elapsed() time.Duration {
	if !tt.started {
		return 0 * time.Second
	}
	return tt.getTime().Sub(tt.startTime)
}
