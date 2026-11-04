package template

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Acceptance tests run without a TTY, so prompts are covered here.
func TestPromptForValuesWithWorkspace(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		answers string
		prompts string
		values  map[string]any
	}{
		{
			name: "workspace context",
			schema: `{"properties": {"jobs_url": {
				"type": "string",
				"description": "Jobs page on {{ .workspace.Hostname }}",
				"default": "{{ .workspace.URL \"/jobs\" }}"
			}}}`,
			answers: "\n",
			prompts: "Jobs page on myhost.test [https://myhost.test/jobs]: ",
			values:  map[string]any{"jobs_url": "https://myhost.test/jobs"},
		},
		{
			name: "declared workspace input",
			schema: `{"properties": {
				"workspace": {"type": "string", "description": "Workspace name", "default": "team-dev", "order": 1},
				"message": {"type": "string", "description": "Message for {{ .workspace }}", "default": "Deploying to {{ .workspace }}", "order": 2}
			}}`,
			answers: "team-prod\n\n",
			prompts: "Workspace name [team-dev]: Message for team-prod [Deploying to team-prod]: ",
			values:  map[string]any{"workspace": "team-prod", "message": "Deploying to team-prod"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, tst := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
			ctx = cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{
				Config: &workspaceConfig.Config{Host: "https://myhost.test"},
			})

			dir := t.TempDir()
			testutil.WriteFile(t, filepath.Join(dir, schemaFileName), tc.schema)
			c, err := newConfig(ctx, os.DirFS(dir), schemaFileName)
			require.NoError(t, err)
			r, err := newRenderer(ctx, c.values, loadHelpers(ctx), os.DirFS(dir), templateDirName, libraryDirName)
			require.NoError(t, err)
			r.workspace, err = loadWorkspaceContext(ctx, c.schema)
			require.NoError(t, err)

			var prompts bytes.Buffer
			copied := make(chan struct{})
			go func() {
				_, _ = io.Copy(&prompts, tst.Stderr)
				close(copied)
			}()
			go func() {
				_, _ = tst.Stdin.WriteString(tc.answers)
				_ = tst.Stdin.Flush()
			}()

			require.NoError(t, c.promptForValues(r))
			tst.Done()
			<-copied

			assert.Equal(t, tc.prompts, prompts.String())
			assert.Equal(t, tc.values, c.values)
		})
	}
}

// Acceptance tests can't use an adb-<id> hostname, because the test server is reachable only
// on localhost.
func TestWorkspaceURLKeepsIDOnHostnameWithID(t *testing.T) {
	ws := workspaceContext{host: "https://adb-111.4.azuredatabricks.net", workspaceID: "111", urlWorkspaceID: "111"}

	// Strict profile matching reads the ID from the query, not from the hostname.
	host, err := ws.URL()
	require.NoError(t, err)
	assert.Equal(t, "https://adb-111.4.azuredatabricks.net?w=111", host)

	jobs, err := ws.URL("/jobs")
	require.NoError(t, err)
	assert.Equal(t, "https://adb-111.4.azuredatabricks.net/jobs?w=111", jobs)
}
