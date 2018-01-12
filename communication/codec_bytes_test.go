package communication

import (
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCodecBytesInterface(t *testing.T) {
	var _ Codec = NewCodecBytes()
}

func TestCodecBytesPack(t *testing.T) {
	table := []struct {
		payload       interface{}
		expectedData  []byte
		expectedError error
	}{
		{`hello`, []byte(`hello`), nil},
		{`hello "name"`, []byte(`hello "name"`), nil},
		{nil, []byte{}, nil},
		{true, nil, errors.New("Cant pack payload: true")},
	}

	codec := codecBytes{}
	for _, tt := range table {
		data, err := codec.Pack(tt.payload)

		if tt.expectedError != nil {
			assert.Error(t, err)
			assert.EqualError(t, err, tt.expectedError.Error())
		} else {
			assert.NoError(t, err)
			assert.Exactly(t, tt.expectedData, data)
		}
	}
}

func TestCodecBytesUnpackToString(t *testing.T) {
	codec := codecBytes{}

	var payload string
	err := codec.Unpack([]byte(`hello`), &payload)

	assert.Error(t, err)
	assert.EqualError(t, err, "Cant unpack to payload: *string")
}

func TestCodecBytesUnpackToByte(t *testing.T) {
	codec := codecBytes{}

	var payload []byte
	err := codec.Unpack([]byte(`hello`), &payload)

	assert.NoError(t, err)
	assert.Exactly(t, []byte(`hello`), payload)
}

func TestCodecBytesUnpack(t *testing.T) {
	table := []struct {
		data            []byte
		expectedPayload []byte
	}{
		{[]byte(`hello`), []byte(`hello`)},
		{[]byte(`hello "name"`), []byte(`hello "name"`)},
		{[]byte{}, []byte{}},
	}

	codec := codecBytes{}
	for _, tt := range table {
		var payload []byte
		err := codec.Unpack(tt.data, &payload)

		assert.NoError(t, err)
		assert.Exactly(t, tt.expectedPayload, payload)
	}
}
