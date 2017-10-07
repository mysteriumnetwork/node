package dto

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	locationLTTelia = Location{"XX", "YY", "AS123"}
	locationJson    = `{
		"country": "XX",
		"city": "YY",
		"asn":"AS123"
	}`
)

func TestLocationSerialize(t *testing.T) {
	jsonBytes, err := json.Marshal(locationLTTelia)
	assert.Nil(t, err)
	assert.JSONEq(t, locationJson, string(jsonBytes))
}

func TestLocationUnserialize(t *testing.T) {
	var model Location

	err := json.Unmarshal([]byte(locationJson), &model)
	assert.Nil(t, err)
	assert.Equal(t, locationLTTelia, model)
}
