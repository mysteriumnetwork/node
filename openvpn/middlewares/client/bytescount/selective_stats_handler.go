package bytescount

import "github.com/pkg/errors"

func NewSelectiveStatsHandler(handler SessionStatsHandler, times int) (SessionStatsHandler, error) {
	if times <= 0 {
		return nil, errors.New("Invalid 'times' parameter")
	}
	delayLeft := 0
	return func(sessionStats SessionStats) error {
		if delayLeft == 0 {
			delayLeft = times - 1
			return handler(sessionStats)
		} else {
			delayLeft--
		}
		return nil
	}, nil
}
