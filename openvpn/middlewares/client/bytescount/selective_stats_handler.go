package bytescount

import "github.com/pkg/errors"

func NewSelectiveStatsHandler(handler SessionStatsHandler, times int) (SessionStatsHandler, error) {
	if times <= 0 {
		return nil, errors.New("Invalid 'times' parameter")
	}
	skipped := 0
	return func(sessionStats SessionStats) error {
		if skipped == times-1 {
			skipped = 0
			return handler(sessionStats)
		} else {
			skipped++
		}
		return nil
	}, nil
}
