package ucm

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	ucmpkg "github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/phases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeUcmYml clones the canonical valid fixture into a temp dir and
// overwrites ucm.yml with body. Returns the workDir so the caller can chain
// runVerbInDir or load it into a *Ucm directly.
func writeUcmYml(t *testing.T, body string) string {
	t.Helper()
	work := cloneFixture(t, validFixtureDir(t))
	require.NoError(t, os.WriteFile(filepath.Join(work, "ucm.yml"), []byte(body), 0o644))
	return work
}

func TestCmd_Summary_HeaderRendersNameAndWorkspace(t *testing.T) {
	stdout, stderr, err := runVerb(t, validFixtureDir(t), "summary")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Name: fixture-valid")
	assert.Contains(t, stdout, "Target: default")
	assert.Contains(t, stdout, "Workspace:")
	assert.Contains(t, stdout, "Host: https://example.cloud.databricks.com")
}

// TestCmd_Summary_NoResourceGroupsWhenEmpty covers the "no deployed resources"
// equivalent for the config-driven view: empty groups do not emit a header,
// and the run still succeeds.
func TestCmd_Summary_NoResourceGroupsWhenEmpty(t *testing.T) {
	work := writeUcmYml(t, `ucm:
  name: empty-deployment

workspace:
  host: https://workspace.cloud.databricks.com
`)

	stdout, _, err := runVerbInDir(t, work, "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Name: empty-deployment")
	assert.NotContains(t, stdout, "Catalogs:")
	assert.NotContains(t, stdout, "Schemas:")
	assert.NotContains(t, stdout, "Storage credentials:")
}

func TestCmd_Summary_ListsCatalogsAndSchemasWithURLs(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Catalogs:")
	assert.Contains(t, stdout, "team_alpha:")
	assert.Contains(t, stdout, "Name: team_alpha")
	assert.Contains(t, stdout, "URL:  https://example.cloud.databricks.com/explore/data/team_alpha")
	assert.Contains(t, stdout, "Schemas:")
	assert.Contains(t, stdout, "bronze:")
	assert.Contains(t, stdout, "Name: team_alpha.bronze")
	assert.Contains(t, stdout, "URL:  https://example.cloud.databricks.com/explore/data/team_alpha/bronze")
}

func TestCmd_Summary_ListsGrantsWithoutURL(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Grants:")
	assert.Contains(t, stdout, "alpha_read:")
	assert.Contains(t, stdout, "Name: catalog team_alpha -> alpha-readers")
	// Grants deliberately do not carry a workspace URL.
	assert.NotContains(t, stdout, "URL:  https://example.cloud.databricks.com/explore/grants")
}

func TestCmd_Summary_ListsStorageCredentials(t *testing.T) {
	work := writeUcmYml(t, `ucm:
  name: creds-only

workspace:
  host: https://workspace.cloud.databricks.com

resources:
  storage_credentials:
    sales_cred:
      name: sales_cred
      aws_iam_role:
        role_arn: arn:aws:iam::123:role/sales
`)

	stdout, _, err := runVerbInDir(t, work, "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Storage credentials:")
	assert.Contains(t, stdout, "sales_cred:")
	assert.Contains(t, stdout, "Name: sales_cred")
	assert.Contains(t, stdout, "URL:  https://workspace.cloud.databricks.com/explore/storage-credentials/sales_cred")
}

// TestCmd_Summary_OutputJSONEmitsConfig exercises the JSON branch. Cobra's
// persistent --output flag only ships when building the root.New() tree, so
// the test loads the fixture directly and replays the Marshal call the RunE
// uses.
func TestCmd_Summary_OutputJSONEmitsConfig(t *testing.T) {
	work := writeUcmYml(t, `ucm:
  name: json-deploy

workspace:
  host: https://workspace.cloud.databricks.com

resources:
  catalogs:
    sales:
      name: sales_prod
`)

	ctx := logdiag.InitContext(context.Background())
	u, err := ucmpkg.Load(ctx, work)
	require.NoError(t, err)
	require.NotNil(t, u)

	phases.LoadDefaultTarget(ctx, u)
	require.False(t, logdiag.HasError(ctx))

	buf, err := json.MarshalIndent(u.Config, "", "  ")
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(buf, &got))

	ucm, ok := got["ucm"].(map[string]any)
	require.True(t, ok, "expected ucm block in JSON output")
	assert.Equal(t, "json-deploy", ucm["name"])

	resources, ok := got["resources"].(map[string]any)
	require.True(t, ok, "expected resources block in JSON output")
	catalogs, ok := resources["catalogs"].(map[string]any)
	require.True(t, ok, "expected catalogs in JSON output")
	assert.Contains(t, catalogs, "sales")
}

// TestRenderSummaryText_EmitsOnlyNonEmptyGroups covers the shape contract:
// groups with no entries do not emit a header.
func TestRenderSummaryText_EmitsOnlyNonEmptyGroups(t *testing.T) {
	work := writeUcmYml(t, `ucm:
  name: demo

workspace:
  host: https://workspace.cloud.databricks.com

resources:
  catalogs:
    sales:
      name: sales_prod
`)

	ctx := logdiag.InitContext(context.Background())
	u, err := ucmpkg.Load(ctx, work)
	require.NoError(t, err)
	phases.LoadDefaultTarget(ctx, u)
	require.False(t, logdiag.HasError(ctx))

	var buf bytes.Buffer
	renderSummaryText(&buf, &u.Config)

	out := buf.String()
	assert.Contains(t, out, "Catalogs:")
	assert.NotContains(t, out, "Schemas:")
	assert.NotContains(t, out, "Grants:")
	assert.NotContains(t, out, "Storage credentials:")
}
