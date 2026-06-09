package root

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldNotify(t *testing.T) {
	// The enabled baseline: interactive text output, released build, not CI, not on DBR.
	enabled := notifyConditions{textOutput: true, isTTY: true}

	tests := []struct {
		name string
		c    notifyConditions
		want bool
	}{
		{"enabled", enabled, true},
		{"on databricks runtime", with(enabled, func(c *notifyConditions) { c.onRuntime = true }), false},
		{"json output", with(enabled, func(c *notifyConditions) { c.textOutput = false }), false},
		{"not a tty", with(enabled, func(c *notifyConditions) { c.isTTY = false }), false},
		{"ci", with(enabled, func(c *notifyConditions) { c.isCI = true }), false},
		{"nothing set", notifyConditions{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldNotify(tt.c))
		})
	}
}

func with(c notifyConditions, mut func(*notifyConditions)) notifyConditions {
	mut(&c)
	return c
}

func TestUpgradeNoticeMessage(t *testing.T) {
	got := upgradeNoticeMessage("0.230.0", "v0.245.0", "https://github.com/databricks/cli/releases/tag/v0.245.0")
	want := "\nA new release of the Databricks CLI is available: 0.230.0 → 0.245.0\n" +
		"https://github.com/databricks/cli/releases/tag/v0.245.0"
	assert.Equal(t, want, got)
}

func TestTrimV(t *testing.T) {
	assert.Equal(t, "0.245.0", trimV("v0.245.0"))
	assert.Equal(t, "0.245.0", trimV("0.245.0"))
	assert.Empty(t, trimV(""))
}
