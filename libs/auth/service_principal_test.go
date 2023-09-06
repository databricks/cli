package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsServicePrincipal_ValidUUID(t *testing.T) {
	userId := "8b948b2e-d2b5-4b9e-8274-11b596f3b652"
	isSP := IsServicePrincipal(userId)
	assert.True(t, isSP, "Expected user ID to be recognized as a service principal")
}

func TestIsServicePrincipal_InvalidUUID(t *testing.T) {
	userId := "invalid"
	isSP := IsServicePrincipal(userId)
	assert.False(t, isSP, "Expected user ID to not be recognized as a service principal")
}
