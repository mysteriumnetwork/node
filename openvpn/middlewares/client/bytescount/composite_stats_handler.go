package bytescount

// NewCompositeStatsHandler composes multiple stats handlers into single one, which executes all handlers sequentially
func NewCompositeStatsHandler(statsHandlers ...SessionStatsHandler) SessionStatsHandler {
	return func(stats SessionStats) error {
		for _, handler := range statsHandlers {
			err := handler(stats)
			if err != nil {
				return err
			}
		}
		return nil
	}
}
