package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargetEmpty(t *testing.T) {
	b := loadTarget(t, "./target_empty", "development")
	assert.Equal(t, "development", b.Config.Bundle.Target)
}
