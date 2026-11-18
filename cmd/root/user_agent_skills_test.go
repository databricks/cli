package root

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
)

func TestParseSkills(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty", "", nil},
		{"single", "databricks-core", []string{"databricks-core"}},
		{"multiple", "a,b,c", []string{"a", "b", "c"}},
		{"trims and lowercases", " Databricks-Core , JOBS ", []string{"databricks-core", "jobs"}},
		{"drops empty entries", ",,a,,", []string{"a"}},
		{"allows underscore and dot", "a_b.c-d", []string{"a_b.c-d"}},
		{"drops characters the user agent rejects", "ok,bad name,a:b,a/b,a+b", []string{"ok"}},
		{"drops leading punctuation", "-a,.b,_c", nil},
		{"dedupes preserving first occurrence", "b,a,b,a,c", []string{"b", "a", "c"}},
		{"dedupes after normalization", "Core,core, CORE", []string{"core"}},
		{"drops entry longer than 64 bytes", strings.Repeat("a", 65) + ",ok", []string{"ok"}},
		{"keeps entry of exactly 64 bytes", strings.Repeat("a", 64), []string{strings.Repeat("a", 64)}},
		{"caps at 20 entries", strings.Join(skillNames(25), ","), skillNames(20)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseSkills(t.Context(), tt.raw))
		})
	}
}

// skillNames returns n distinct valid skill names.
func skillNames(n int) []string {
	names := make([]string, 0, n)
	for i := range n {
		names = append(names, "skill"+strings.Repeat("x", i))
	}
	return names
}

func TestWithSkillsInUserAgent(t *testing.T) {
	tests := []struct {
		name             string
		skills           string
		disableTelemetry string
		want             string
	}{
		{"unset", "", "", ""},
		{"set", "databricks-core,databricks-jobs", "", "skill/databricks-core skill/databricks-jobs"},
		{"telemetry disabled", "databricks-core", "1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Always set both: an empty value in the context overrides the host environment.
			ctx := env.Set(t.Context(), skillsInvokedEnvVar, tt.skills)
			ctx = env.Set(ctx, "DATABRICKS_CLI_DISABLE_TELEMETRY", tt.disableTelemetry)
			ua := useragent.FromContext(withSkillsInUserAgent(ctx))
			if tt.want == "" {
				assert.NotContains(t, ua, "skill/")
			} else {
				assert.Contains(t, ua, tt.want)
			}
		})
	}
}
