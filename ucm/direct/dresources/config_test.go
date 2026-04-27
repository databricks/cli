package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustLoadConfig(t *testing.T) {
	// Resources start empty until the resource-struct migration lands.
	cfg := MustLoadConfig()
	assert.NotNil(t, cfg)
}

func TestGetResourceConfig(t *testing.T) {
	// Nonexistent resources return an empty config without panicking.
	assert.Empty(t, GetResourceConfig("nonexistent").RecreateOnChanges)
}
