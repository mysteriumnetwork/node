package utils

import "time"

// SettableClock allows settings and getting time, which is convenient for testing
type SettableClock struct {
	time time.Time
}

// SetTime sets time to be returned from GetTime
func (fc *SettableClock) SetTime(time time.Time) {
	fc.time = time
}

// GetTime returns set time
func (fc *SettableClock) GetTime() time.Time {
	return fc.time
}
