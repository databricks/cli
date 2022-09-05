package tests

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func SetTestEnv(t *testing.T, key, value string) {
	originalValue, isSet := os.LookupEnv(key)

	err := os.Setenv(key, value)
	assert.NoError(t, err)

	t.Cleanup(func() {
		if isSet {
			os.Setenv(key, originalValue)
		} else {
			os.Unsetenv(key)
		}
	})
}
