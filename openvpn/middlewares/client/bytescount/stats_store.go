package bytescount

// SessionStatsStore keeps session stats
type SessionStatsStore struct {
	sessionStats SessionStats
}

// Save saves session stats to store
func (store *SessionStatsStore) Save(stats SessionStats) {
	store.sessionStats = stats
}

// Retrieve retrieves session stats from store
func (store *SessionStatsStore) Retrieve() SessionStats {
	return store.sessionStats
}
