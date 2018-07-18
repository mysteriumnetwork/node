package utils

import "fmt"

type ByteArrayString struct {
	array []byte
}

func ToByteArrayString(val []byte) *ByteArrayString {
	return &ByteArrayString{
		array: val,
	}
}

func (bas *ByteArrayString) MarshalJSON() ([]byte, error) {

	return []byte(fmt.Sprintf(`"0x%x"`, bas.array)), nil
}
