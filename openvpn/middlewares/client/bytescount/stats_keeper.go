package bytescount

// SessionStatsKeeper keeps session stats
type SessionStatsKeeper struct {
	sessionStats SessionStats
}

// Save saves session stats to keeper
func (keeper *SessionStatsKeeper) Save(stats SessionStats) {
	keeper.sessionStats = stats
}

// Retrieve retrieves session stats from keeper
func (keeper *SessionStatsKeeper) Retrieve() SessionStats {
	return keeper.sessionStats
}
