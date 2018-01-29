package bytescount

import (
	"errors"
	"time"
)

// SessionStatsKeeper keeps session stats
type SessionStatsKeeper interface {
	Save(stats SessionStats)
	Retrieve() SessionStats
	MarkSessionStart()
	GetSessionDuration() (time.Duration, error)
}

// Clock function returns current time
type Clock func() time.Time

type sessionStatsKeeper struct {
	sessionStats SessionStats
	clock        Clock
	sessionStart *time.Time
}

// NewSessionStatsKeeper returns new session stats keeper with given clock function
func NewSessionStatsKeeper(clock Clock) SessionStatsKeeper {
	return &sessionStatsKeeper{clock: clock}
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
	time := keeper.clock()
	keeper.sessionStart = &time
}

// GetSessionDuration returns elapsed time from marked session start
func (keeper *sessionStatsKeeper) GetSessionDuration() (time.Duration, error) {
	if keeper.sessionStart == nil {
		return time.Duration(0), errors.New("session start was not marked")
	}
	duration := keeper.clock().Sub(*keeper.sessionStart)
	return duration, nil
}
