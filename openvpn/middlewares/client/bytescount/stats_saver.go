package bytescount

// NewSessionStatsSaver returns stats handler, which saves stats to global stats store
func NewSessionStatsSaver(statsStore *SessionStatsStore) SessionStatsHandler {
	return func(sessionStats SessionStats) error {
		statsStore.Save(sessionStats)
		return nil
	}
}
