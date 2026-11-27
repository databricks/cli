package skills

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

// Each case sets its variables in the context, with "" for unset. env.Get
// reads the context first, so an exported DATABRICKS_SKILLS_PREVIEW,
// DATABRICKS_SKILLS_HINTS or DATABRICKS_AGENT_FEATURES_DISABLED on the
// developer's machine does not leak in.
func TestEnabled(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false},
		{name: "true", value: "true", want: true},
		{name: "one", value: "1", want: true},
		{name: "false", value: "false", want: false},
		{name: "zero", value: "0", want: false},
		{name: "empty", value: "", want: false},
		{name: "invalid", value: "maybe", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), AgentFeaturesDisabledEnv, "")
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.value)
			assert.Equal(t, c.want, Enabled(ctx))
		})
	}
}

// Only a true value turns DATABRICKS_AGENT_FEATURES_DISABLED on. Unset, false
// and an invalid value leave the feature as its own switches set it.
func TestAgentFeaturesDisabled(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "unset", value: "", want: false},
		{name: "true", value: "true", want: true},
		{name: "one", value: "1", want: true},
		{name: "yes", value: "yes", want: true},
		{name: "false", value: "false", want: false},
		{name: "zero", value: "0", want: false},
		{name: "invalid", value: "maybe", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), AgentFeaturesDisabledEnv, c.value)
			assert.Equal(t, c.want, AgentFeaturesDisabled(ctx))
		})
	}
}

// With both skills switches on, a true DATABRICKS_AGENT_FEATURES_DISABLED turns
// Enabled and HintsEnabled off. Any other value leaves them on.
func TestAgentFeaturesDisabledOverridesSkillsSwitches(t *testing.T) {
	cases := []struct {
		name     string
		disabled string
		want     bool
	}{
		{name: "unset", want: true},
		{name: "true", disabled: "true", want: false},
		{name: "one", disabled: "1", want: false},
		{name: "false", disabled: "false", want: true},
		{name: "invalid", disabled: "maybe", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), AgentFeaturesDisabledEnv, c.disabled)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", "1")
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")
			assert.Equal(t, c.want, Enabled(ctx))
			assert.Equal(t, c.want, HintsEnabled(ctx))
		})
	}
}

func TestHintsEnabled(t *testing.T) {
	cases := []struct {
		name    string
		preview string
		hints   string
		want    bool
	}{
		{name: "both unset", want: false},
		{name: "preview off, hints on", preview: "false", hints: "true", want: false},
		{name: "preview unset, hints on", hints: "true", want: false},
		{name: "preview on, hints unset", preview: "true", want: false},
		{name: "preview on, hints false", preview: "true", hints: "false", want: false},
		{name: "preview on, hints invalid", preview: "true", hints: "maybe", want: false},
		{name: "preview on, hints true", preview: "true", hints: "true", want: true},
		{name: "preview on, hints one", preview: "1", hints: "1", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), AgentFeaturesDisabledEnv, "")
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.preview)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", c.hints)
			assert.Equal(t, c.want, HintsEnabled(ctx))
		})
	}
}
