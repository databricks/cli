package tags

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAwsNormalizeKey(t *testing.T) {
	assert.Equal(t, "1 a b c", awsTag.NormalizeKey("1 a b c"))
	assert.Equal(t, "+-=.:/@__", awsTag.NormalizeKey("+-=.:/@?)"))
	assert.Equal(t, "test", awsTag.NormalizeKey("test"))

	// Remove marks; unicode becomes underscore.
	assert.Equal(t, "cafe _", awsTag.NormalizeKey("café 🍎"))

	// Replace forbidden characters with underscore.
	assert.Equal(t, "cafe __", awsTag.NormalizeKey("café 🍎?"))
}

func TestAwsNormalizeValue(t *testing.T) {
	assert.Equal(t, "1 a b c", awsTag.NormalizeValue("1 a b c"))
	assert.Equal(t, "+-=.:/@__", awsTag.NormalizeValue("+-=.:/@?)"))
	assert.Equal(t, "test", awsTag.NormalizeValue("test"))

	// Remove marks; unicode becomes underscore.
	assert.Equal(t, "cafe _", awsTag.NormalizeValue("café 🍎"))

	// Replace forbidden characters with underscore.
	assert.Equal(t, "cafe __", awsTag.NormalizeValue("café 🍎?"))
}

func TestAwsValidateKey(t *testing.T) {
	assert.ErrorContains(t, awsTag.ValidateKey(""), "not be empty")
	assert.ErrorContains(t, awsTag.ValidateKey(strings.Repeat("a", 512)), "length")
	assert.ErrorContains(t, awsTag.ValidateKey("café 🍎"), "latin")
	assert.ErrorContains(t, awsTag.ValidateKey("????"), "pattern")
	assert.NoError(t, awsTag.ValidateKey(strings.Repeat("a", 127)))
	assert.NoError(t, awsTag.ValidateKey(awsTag.NormalizeKey("café 🍎")))
}

func TestAwsValidateValue(t *testing.T) {
	assert.ErrorContains(t, awsTag.ValidateValue(strings.Repeat("a", 512)), "length")
	assert.ErrorContains(t, awsTag.ValidateValue("café 🍎"), "latin1")
	assert.ErrorContains(t, awsTag.ValidateValue("????"), "pattern")
	assert.NoError(t, awsTag.ValidateValue(strings.Repeat("a", 127)))
	assert.NoError(t, awsTag.ValidateValue(awsTag.NormalizeValue("café 🍎")))
}
