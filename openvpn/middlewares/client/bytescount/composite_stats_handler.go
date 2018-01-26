package bytescount

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
