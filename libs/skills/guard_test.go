package skills

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestEnabled(t *testing.T) {
	cases := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset", want: false},
		{name: "true", set: true, value: "true", want: true},
		{name: "one", set: true, value: "1", want: true},
		{name: "false", set: true, value: "false", want: false},
		{name: "zero", set: true, value: "0", want: false},
		{name: "empty", set: true, value: "", want: false},
		{name: "invalid", set: true, value: "maybe", want: false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			if c.set {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.value)
			}
			assert.Equal(t, c.want, Enabled(ctx))
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
			ctx := t.Context()
			if c.preview != "" {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.preview)
			}
			if c.hints != "" {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", c.hints)
			}
			assert.Equal(t, c.want, HintsEnabled(ctx))
		})
	}
}
