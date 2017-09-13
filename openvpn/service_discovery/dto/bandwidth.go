package dto

import (
	"github.com/mysterium/node/datasize"
)

// Speed in b/s (bits per second )
// 64 b/s = 8 B/s (since there are 8 bits in a byte)
type Bandwidth datasize.BitSize
