package tags

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAzureNormalizeKey(t *testing.T) {
	assert.Equal(t, "test", azureTag.NormalizeKey("test"))
	assert.Equal(t, "café __", azureTag.NormalizeKey("café 🍎?"))
}

func TestAzureNormalizeValue(t *testing.T) {
	assert.Equal(t, "test", azureTag.NormalizeValue("test"))
	assert.Equal(t, "café _?", azureTag.NormalizeValue("café 🍎?"))
}

func TestAzureValidateKey(t *testing.T) {
	assert.ErrorContains(t, azureTag.ValidateKey(""), "not be empty")
	assert.ErrorContains(t, azureTag.ValidateKey(strings.Repeat("a", 513)), "length")
	assert.ErrorContains(t, azureTag.ValidateKey("café 🍎"), "latin")
	assert.ErrorContains(t, azureTag.ValidateKey("????"), "pattern")
	assert.NoError(t, azureTag.ValidateKey(strings.Repeat("a", 127)))
	assert.NoError(t, azureTag.ValidateKey(azureTag.NormalizeKey("café 🍎")))
}

func TestAzureValidateValue(t *testing.T) {
	assert.ErrorContains(t, azureTag.ValidateValue(strings.Repeat("a", 513)), "length")
	assert.ErrorContains(t, azureTag.ValidateValue("café 🍎"), "latin")
	assert.NoError(t, azureTag.ValidateValue(strings.Repeat("a", 127)))
	assert.NoError(t, azureTag.ValidateValue(azureTag.NormalizeValue("café 🍎")))
}
