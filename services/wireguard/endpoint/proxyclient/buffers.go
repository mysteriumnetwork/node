package proxyclient

import (
	"sync"
)

type BufferPool struct {
	sp sync.Pool
}

func NewBufferPool(size int) (pool *BufferPool) {
	return &BufferPool{
		sp: sync.Pool{
			New: func() interface{} {
				return make([]byte, size, size) // buffer don't grow
			},
		},
	}
}

func (pool *BufferPool) Get() (buffer []byte) {
	return pool.sp.Get().([]byte)
}

func (pool *BufferPool) Put(buffer []byte) {
	// not necessary to clean buffer
	pool.sp.Put(buffer)
}
