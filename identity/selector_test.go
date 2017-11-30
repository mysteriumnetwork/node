package identity

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

var dir = "/tmp"

func TestSelectEmptyAndCreateNewIdentity(t *testing.T) {
	id := ""

	identity, err := SelectIdentity(dir, id)

	assert.Nil(t, err)
	assert.Len(t, *identity, 42)
}

func TestMissingIdentity(t *testing.T) {
	id := "identity"
	_, err := SelectIdentity(dir, id)

	assert.Error(t, err, "identity doesn't exist")
}
