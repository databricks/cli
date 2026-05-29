package lakebox

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateNameAcceptsAscii(t *testing.T) {
	require.NoError(t, validateName(""))
	require.NoError(t, validateName("my-project"))
	require.NoError(t, validateName(strings.Repeat("a", 256))) // boundary: exactly the limit
}

func TestValidateNameRejectsOversize(t *testing.T) {
	err := validateName(strings.Repeat("a", 257))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "257 bytes")
	assert.Contains(t, err.Error(), "256")
}

func TestValidateNameCountsBytesNotRunes(t *testing.T) {
	// 64 panda emoji = 64 × 4 bytes = 256 bytes — at the limit, OK.
	require.NoError(t, validateName(strings.Repeat("🐼", 64)))
	// 65 = 260 bytes, rejected.
	err := validateName(strings.Repeat("🐼", 65))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "260 bytes")
}
