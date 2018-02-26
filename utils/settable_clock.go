package utils

import "time"

// SettableClock allows settings and getting time, which is convenient for testing
type SettableClock struct {
	time time.Time
}

// SetTime sets time to be returned from GetTime
func (clock *SettableClock) SetTime(time time.Time) {
	clock.time = time
}

// GetTime returns set time
func (clock *SettableClock) GetTime() time.Time {
	return clock.time
}

// AddTime adds given duration to current clock time
func (clock *SettableClock) AddTime(duration time.Duration) {
	newTime := clock.GetTime().Add(duration)
	clock.SetTime(newTime)
}
