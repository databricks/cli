package ai

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderStatus renders the status template against the JSON envelope, exactly as
// the command does, so the test covers the real template branches.
func renderStatus(t *testing.T, data statusData) string {
	t.Helper()
	tmpl, err := template.New("status").Parse(statusTemplate)
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, envelope{V: envelopeVersion, Data: data}))
	return buf.String()
}

func TestStatusTemplateSingleRun(t *testing.T) {
	out := renderStatus(t, statusData{
		RunID:        "123",
		Status:       "RUNNING",
		User:         "me@example.com",
		DashboardURL: "https://example.test/run/123",
	})
	assert.Contains(t, out, "Run ID:       123")
	assert.Contains(t, out, "Status:       RUNNING")
	assert.Contains(t, out, "User:")
	assert.Contains(t, out, "me@example.com")
	assert.Contains(t, out, "Dashboard:    https://example.test/run/123")
	assert.NotContains(t, out, "Sweep")
}

func TestYAMLConfigPath(t *testing.T) {
	// No tasks, or a task without GenAiComputeTask, yields no path.
	assert.Equal(t, "", yamlConfigPath(&jobs.Run{}))
	assert.Equal(t, "", yamlConfigPath(&jobs.Run{Tasks: []jobs.RunTask{{}}}))

	run := &jobs.Run{Tasks: []jobs.RunTask{{
		GenAiComputeTask: &jobs.GenAiComputeTask{YamlParametersFilePath: "/Workspace/cfg.yaml"},
	}}}
	assert.Equal(t, "/Workspace/cfg.yaml", yamlConfigPath(run))
}

func TestStatusTemplateSweep(t *testing.T) {
	out := renderStatus(t, statusData{
		RunID:  "456",
		Status: "RUNNING",
		Sweep: &sweepInfo{
			Total: 4, Completed: 2, Succeeded: 1, Failed: 1, Active: 2,
			Tasks: []sweepTask{
				{TaskKey: "iter_0", RunID: "789", Status: "SUCCESS", Experiment: "my-exp"},
			},
		},
	})
	assert.Contains(t, out, "Sweep Run ID: 456")
	assert.Contains(t, out, "Total:        4")
	assert.Contains(t, out, "Sweep Tasks:")
	assert.Contains(t, out, "iter_0")
	assert.Contains(t, out, "my-exp")
	// The single-run rows must not appear in the sweep view.
	assert.NotContains(t, out, "Dashboard:")
}
