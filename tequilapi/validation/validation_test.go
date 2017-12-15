package validation

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestErrorsListRenderedInJson(t *testing.T) {
	errorMap := NewErrorMap()
	errorMap.ForField("email").AddError("required", "field required")
	errorMap.ForField("email").AddError("another", "error")
	errorMap.ForField("username").AddError("invalid", "field invalid")

	v, err := json.Marshal(errorMap)
	assert.Nil(t, err)

	assert.JSONEq(
		t,
		`{
			"email" : [
				{ "code" : "required" , "message" : "field required" },
				{ "code" : "another" , "message" : "error"}
			],
			"username" : [
				{ "code" : "invalid" , "message" : "field invalid" }
			]
		}`,
		string(v))

}
