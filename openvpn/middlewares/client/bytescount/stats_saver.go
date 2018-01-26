package bytescount

// NewSessionStatsSaver returns stats handler, which saves stats stats keeper
func NewSessionStatsSaver(statsKeeper *SessionStatsKeeper) SessionStatsHandler {
	return func(sessionStats SessionStats) error {
		statsKeeper.Save(sessionStats)
		return nil
	}
}
