package connection

import "sync"

func applyOnce(f func()) func() {
	once := sync.Once{}
	return func() {
		once.Do(f)
	}
}
