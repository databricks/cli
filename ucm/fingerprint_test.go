package ucm_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/stretchr/testify/assert"
)

func TestUserFingerprintIsEmptyWhenZero(t *testing.T) {
	f := ucm.UserFingerprint{}
	assert.True(t, f.IsEmpty())
}

func TestUserFingerprintIsNotEmptyWhenHostSet(t *testing.T) {
	f := ucm.UserFingerprint{Host: "h"}
	assert.False(t, f.IsEmpty())
}
