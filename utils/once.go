package utils

import "sync"

// CallOnce function decorator returns new function which can be called only once and is thread safe
func CallOnce(f func()) func() {
	once := sync.Once{}
	return func() {
		once.Do(f)
	}
}
