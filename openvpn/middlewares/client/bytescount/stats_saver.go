package bytescount

// NewSessionStatsSaver returns stats handler, which saves stats to global stats store
func NewSessionStatsSaver() SessionStatsHandler {
	return func(sessionStats SessionStats) error {
		GetSessionStatsStore().Set(sessionStats)
		return nil
	}
}
