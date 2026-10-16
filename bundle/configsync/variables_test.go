package configsync

import (
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestConfig(t *testing.T, yaml string) *config.Root {
	t.Helper()
	root, diags := config.LoadFromBytes("test.yml", []byte(yaml))
	require.NoError(t, diags.Error())
	return root
}

// scalarView returns a view of the string s.
func scalarView(t *testing.T, s string) structvar.View {
	t.Helper()
	root := loadTestConfig(t, fmt.Sprintf("bundle:\n  name: %q\n", s))
	return root.View().Lookup(structpath.MustParsePath("bundle.name"))
}

// tasksView returns views of job tasks, each given as the YAML of its fields in flow style.
func tasksView(t *testing.T, tasks ...string) []structvar.View {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("resources:\n  jobs:\n    j:\n      tasks:\n")
	for _, task := range tasks {
		fmt.Fprintf(&sb, "        - {%s}\n", task)
	}
	root := loadTestConfig(t, sb.String())
	var out []structvar.View
	for _, elem := range root.View().Lookup(structpath.MustParsePath("resources.jobs.j.tasks")).Sequence() {
		out = append(out, elem)
	}
	return out
}

// variablesConfig returns a resolved configuration with the given variable values.
func variablesConfig(t *testing.T, values map[string]any) resolvedConfig {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("variables:\n")
	for name, value := range values {
		fmt.Fprintf(&sb, "  %s:\n    value: %#v\n", name, value)
	}
	return resolvedConfig{view: loadTestConfig(t, sb.String()).View(), overrides: map[string]any{}}
}

// TestRestoreOriginalRefs_HardcodedFieldNotRewritten fences the Replace safety
// invariant: a hardcoded leaf must never be rewritten to a variable reference
// just because the remote value coincidentally matches a variable elsewhere.
func TestRestoreOriginalRefs_HardcodedFieldNotRewritten(t *testing.T) {
	preResolved := scalarView(t, "us-east-1")
	resolved := variablesConfig(t, map[string]any{"region": "main"})
	// Even though "main" matches ${var.region}, restoreOriginalRefs must NOT
	// rewrite it — the original was hardcoded.
	result := restoreOriginalRefs("main", preResolved, resolved, &RestoreStats{})
	assert.Equal(t, "main", result)
}

// TestRestoreFromSiblings_ValueMatchesVariableButDifferentPath fences the Add
// false-positive guard: a new leaf's value matching a variable at a DIFFERENT
// relative path in a sibling must not trigger restoration.
func TestRestoreFromSiblings_ValueMatchesVariableButDifferentPath(t *testing.T) {
	// Sibling uses ${var.retry_count}=5 at .max_retries. New element has
	// .min_retry_interval=5 — coincidental match at a DIFFERENT relative path.
	siblings := tasksView(t, `task_key: main, max_retries: "${var.retry_count}"`)
	resolved := variablesConfig(t, map[string]any{"retry_count": 5})
	value := map[string]any{
		"task_key":           "other",
		"min_retry_interval": int64(5),
	}
	result := restoreFromSiblings(value, siblings, resolved, &RestoreStats{}).(map[string]any)
	assert.Equal(t, "other", result["task_key"])
	assert.Equal(t, int64(5), result["min_retry_interval"])
}

// TestRestoreFromSiblings_AmbiguousAcrossSiblings fences the multi-variable
// same-value rule: when two siblings use different variables at the same
// relative path that both resolve to the same value, restoration is skipped.
func TestRestoreFromSiblings_IntVariableMatchesInt64Field(t *testing.T) {
	// The variable decodes from YAML as int; the remote field is a typed int64.
	siblings := tasksView(t, `task_key: main, max_retries: "${var.retry_count}"`)
	resolved := variablesConfig(t, map[string]any{"retry_count": 5})
	value := map[string]any{"task_key": "other", "max_retries": int64(5)}
	result := restoreFromSiblings(value, siblings, resolved, &RestoreStats{}).(map[string]any)
	assert.Equal(t, "${var.retry_count}", result["max_retries"])
}

func TestRestoreFromSiblings_AmbiguousAcrossSiblings(t *testing.T) {
	siblings := tasksView(t, `task_key: "${var.landing_schema}"`, `task_key: "${var.curated_schema}"`)
	resolved := variablesConfig(t, map[string]any{"landing_schema": "raw_data", "curated_schema": "raw_data"})
	value := map[string]any{"task_key": "raw_data"}
	result := restoreFromSiblings(value, siblings, resolved, &RestoreStats{}).(map[string]any)
	assert.Equal(t, "raw_data", result["task_key"])
}

// TestRestoreCompoundInterpolation covers the template alignment algorithm.
// End-to-end coverage (pure ref match, sibling match, non-sequence skip, etc.)
// lives in acceptance/bundle/config-remote-sync/resolve_variables.
func TestRestoreCompoundInterpolation(t *testing.T) {
	resolved := variablesConfig(t, map[string]any{
		"host": "dev-sql.example.com",
		"port": "1433",
		"db":   "analytics_dev",
		"acct": "acct",
	})

	tests := []struct {
		name     string
		template string
		remote   string
		want     string
	}{
		{
			name:     "suffix change",
			template: "/mnt/${var.acct}/raw/landing",
			remote:   "/mnt/acct/raw/landing_v2",
			want:     "/mnt/${var.acct}/raw/landing_v2",
		},
		{
			name:     "partial variable change preserves others",
			template: "jdbc:sqlserver://${var.host}:${var.port};database=${var.db}",
			remote:   "jdbc:sqlserver://dev-sql.example.com:5432;database=analytics_dev",
			want:     "jdbc:sqlserver://${var.host}:5432;database=${var.db}",
		},
		{
			name:     "completely unrelated value falls back to hardcoded",
			template: "${var.acct}-phi-encryption-key",
			remote:   "master-encryption-key-v2",
			want:     "master-encryption-key-v2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := restoreOriginalRefs(tt.remote, scalarView(t, tt.template), resolved, &RestoreStats{})
			assert.Equal(t, tt.want, result)
		})
	}
}
