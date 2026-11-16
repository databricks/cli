package cmdio

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptModelNoColor(t *testing.T) {
	m := newPromptModel(PromptOptions{Label: "Name"}, true)
	assert.Contains(t, m.View(), ansiGreen)

	m = newPromptModel(PromptOptions{Label: "Name"}, false)
	v := m.View()
	assert.NotContains(t, v, ansiGreen)
	assert.Contains(t, v, ansiBold)
}
