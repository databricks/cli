package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustLoadConfig(t *testing.T) {
	cfg := MustLoadConfig()
	assert.NotEmpty(t, cfg.Resources)
}

func TestGetResourceConfig(t *testing.T) {
	assert.NotNil(t, GetResourceConfig("volumes"))
	assert.Nil(t, GetResourceConfig("nonexistent"))
}
