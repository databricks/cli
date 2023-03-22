package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLockDefaults(t *testing.T) {
	lock := Lock{}
	assert.True(t, lock.IsEnabled())
}

func TestLockIsEnabled(t *testing.T) {
	lock := Lock{Enabled: new(bool)}

	*lock.Enabled = false
	assert.False(t, lock.IsEnabled())

	*lock.Enabled = true
	assert.True(t, lock.IsEnabled())
}
