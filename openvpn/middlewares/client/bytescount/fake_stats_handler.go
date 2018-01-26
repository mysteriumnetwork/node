package bytescount

type fakeStatsHandler struct {
	LastSessionStats SessionStats
}

func (sender *fakeStatsHandler) save(sessionStats SessionStats) error {
	sender.LastSessionStats = sessionStats
	return nil
}
