package utils

import (
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"testing"
)

func TestWriteAsJsonReturnsExpectedResponse(t *testing.T) {

	respRecorder := httptest.NewRecorder()

	type TestStruct struct {
		IntField    int
		StringField string `json:"renamed"`
	}

	WriteAsJson(TestStruct{1, "abc"}, respRecorder)

	result := respRecorder.Result()

	assert.Equal(t, "application/json", result.Header.Get("Content-type"))
	assert.JSONEq(
		t,
		`{
            "IntField" : 1,
			"renamed" : "abc"
		}`,
		respRecorder.Body.String())

}
