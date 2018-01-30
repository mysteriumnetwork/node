package bytescount

import (
	"time"
)

// SessionStatsKeeper keeps session stats
type SessionStatsKeeper interface {
	Save(stats SessionStats)
	Retrieve() SessionStats
	MarkSessionStart()
	GetSessionDuration() time.Duration
}

// TimeGetter function returns current time
type TimeGetter func() time.Time

type sessionStatsKeeper struct {
	sessionStats SessionStats
	timeGetter   TimeGetter
	sessionStart *time.Time
}

// NewSessionStatsKeeper returns new session stats keeper with given timeGetter function
func NewSessionStatsKeeper(timeGetter TimeGetter) SessionStatsKeeper {
	return &sessionStatsKeeper{timeGetter: timeGetter}
}

// Save saves session stats to keeper
func (keeper *sessionStatsKeeper) Save(stats SessionStats) {
	keeper.sessionStats = stats
}

// Retrieve retrieves session stats from keeper
func (keeper *sessionStatsKeeper) Retrieve() SessionStats {
	return keeper.sessionStats
}

// MarkSessionStart marks current time as session start time for statistics
func (keeper *sessionStatsKeeper) MarkSessionStart() {
	time := keeper.timeGetter()
	keeper.sessionStart = &time
}

// GetSessionDuration returns elapsed time from marked session start
func (keeper *sessionStatsKeeper) GetSessionDuration() time.Duration {
	if keeper.sessionStart == nil {
		return time.Duration(0)
	}
	duration := keeper.timeGetter().Sub(*keeper.sessionStart)
	return duration
}
