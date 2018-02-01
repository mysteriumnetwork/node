package dto

import (
	"github.com/mysterium/node/datasize"
	"strconv"
)

// Speed in b/s (bits per second)
// 64 b/s = 8 B/s (since there are 8 bits in a byte)
type Bandwidth datasize.BitSize

func (value Bandwidth) MarshalJSON() ([]byte, error) {
	valueBits := datasize.BitSize(value).Bits()
	valueJson := strconv.FormatUint(valueBits, 10)

	return []byte(valueJson), nil
}

func (value *Bandwidth) UnmarshalJSON(valueJson []byte) error {
	valueBits, err := strconv.ParseUint(string(valueJson), 10, 64)
	*value = Bandwidth(valueBits)

	return err
}
