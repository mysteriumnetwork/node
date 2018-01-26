package bytescount

type sessionStatsStore struct {
	sessionStats SessionStats
}

func (store *sessionStatsStore) Clear() {
	store.sessionStats = SessionStats{}
}

func (store *sessionStatsStore) Set(stats SessionStats) {
	store.sessionStats = stats
}

func (store *sessionStatsStore) Get() SessionStats {
	return store.sessionStats
}

var globalStatsStore sessionStatsStore

// GetSessionStatsStore returns singleton store, which keeps session stats
func GetSessionStatsStore() *sessionStatsStore {
	return &globalStatsStore
}
