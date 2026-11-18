package root_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillNames returns n distinct skill names of exactly length bytes.
func skillNames(n, length int) []string {
	names := make([]string, 0, n)
	for i := range n {
		prefix := fmt.Sprintf("s%02d", i)
		names = append(names, prefix+strings.Repeat("x", length-len(prefix)))
	}
	return names
}

// pairsSize is the length of " skill/<name>" for every name.
func pairsSize(names []string) int {
	size := 0
	for _, name := range names {
		size += len(" skill/") + len(name)
	}
	return size
}

func TestParseSkills(t *testing.T) {
	// 38-byte names (the longest current skill) make 45-byte pairs: five fill
	// 225 bytes exactly and a sixth doesn't fit.
	fiveFit := skillNames(5, 38)
	sixTooMany := skillNames(6, 38)
	// 64-, 64-, 64- and 22-byte pairs make 214 bytes, so " skill/abcd" (11)
	// lands exactly on 225 and nothing more fits.
	exact := []string{strings.Repeat("a", 57), strings.Repeat("b", 57), strings.Repeat("c", 57), "databricks-core", "abcd"}

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
		{"drops disallowed skill-name characters", "ok,bad name,a:b,a/b,a+b", []string{"ok"}},
		{"drops leading punctuation", "-a,.b,_c", nil},
		{"dedupes preserving first occurrence", "b,a,b,a,c", []string{"b", "a", "c"}},
		{"dedupes after normalization", "Core,core, CORE", []string{"core"}},
		{"drops entry longer than 64 bytes", strings.Repeat("a", 65) + ",ok", []string{"ok"}},
		{"keeps entry of exactly 64 bytes", strings.Repeat("a", 64), []string{strings.Repeat("a", 64)}},
		{"keeps five of the longest skill names", strings.Join(fiveFit, ","), fiveFit},
		{"drops pairs over the budget", strings.Join(sixTooMany, ","), sixTooMany[:5]},
		{"fills the budget exactly", strings.Join(exact, ",") + ",d", exact},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := root.ParseSkills(t.Context(), tt.raw)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, pairsSize(got), root.MaxSkillsUserAgentLength)
		})
	}
}

func TestWithSkillsInUserAgent(t *testing.T) {
	tests := []struct {
		name             string
		env              string
		args             []string
		disableTelemetry string
		want             string
	}{
		{"unset", "", nil, "", ""},
		{"env var", "databricks-core,databricks-jobs", nil, "", "skill/databricks-core skill/databricks-jobs"},
		{"flag", "", []string{"--skills-invoked", "databricks-core,databricks-jobs"}, "", "skill/databricks-core skill/databricks-jobs"},
		{"flag wins over env var", "from-env", []string{"--skills-invoked", "from-flag"}, "", "skill/from-flag"},
		{"empty flag still wins over env var", "from-env", []string{"--skills-invoked="}, "", ""},
		{"telemetry disabled with env var", "databricks-core", nil, "1", ""},
		{"telemetry disabled with flag", "", []string{"--skills-invoked", "databricks-core"}, "1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "test"}
			root.InitSkillsInvokedFlag(cmd)
			require.NoError(t, cmd.ParseFlags(tt.args))

			// Always set both: an empty value in the context overrides the host environment.
			ctx := env.Set(t.Context(), root.SkillsInvokedEnvVar, tt.env)
			ctx = env.Set(ctx, "DATABRICKS_CLI_DISABLE_TELEMETRY", tt.disableTelemetry)
			ua := useragent.FromContext(root.WithSkillsInUserAgent(ctx, cmd))
			if tt.want == "" {
				assert.NotContains(t, ua, "skill/")
			} else {
				assert.Contains(t, ua, tt.want)
			}
		})
	}
}
