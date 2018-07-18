package utils

import (
	"bytes"
	"fmt"
)

type ByteArrayString struct {
	array []byte
}

func ToByteArrayString(val []byte) *ByteArrayString {
	return &ByteArrayString{
		array: val,
	}
}

func (bas *ByteArrayString) MarshalJSON() ([]byte, error) {
	var buff bytes.Buffer
	fmt.Fprint(&buff, `"`)
	for _, val := range bas.array {
		fmt.Fprint(&buff, "\\\\x")
		fmt.Fprintf(&buff, "%02x", val)
	}
	fmt.Fprint(&buff, `"`)
	return buff.Bytes(), nil
}
