package iamutil

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestIsServicePrincipal_ValidUUID(t *testing.T) {
	user := &iam.User{
		UserName: "8b948b2e-d2b5-4b9e-8274-11b596f3b652",
	}
	isSP := IsServicePrincipal(user)
	assert.True(t, isSP, "Expected user ID to be recognized as a service principal")
}

func TestIsServicePrincipal_InvalidUUID(t *testing.T) {
	user := &iam.User{
		UserName: "invalid",
	}
	isSP := IsServicePrincipal(user)
	assert.False(t, isSP, "Expected user ID to not be recognized as a service principal")
}
