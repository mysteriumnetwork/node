package connection

import "sync"

func callOnce(f func()) func() {
	once := sync.Once{}
	return func() {
		once.Do(f)
	}
}
