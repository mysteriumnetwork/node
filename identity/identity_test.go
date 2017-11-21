package identity

import (
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
)

func Test_CreateNewIdentity(t *testing.T) {
	id, err := CreateNewIdentity("testdata")
	assert.NoError(t, err)
	assert.Equal(t, len(id), 42)
}

func Test_GetIdentities(t *testing.T) {
	ids := GetIdentities("testdata")
	for _, id := range ids {
		fmt.Println(id)
	}
}