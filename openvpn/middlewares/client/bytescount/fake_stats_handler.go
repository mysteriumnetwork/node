package bytescount

type fakeStatsRecorder struct {
	LastSessionStats SessionStats
}

func (sender *fakeStatsRecorder) record(sessionStats SessionStats) error {
	sender.LastSessionStats = sessionStats
	return nil
}
