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

// seedTfstate drops a fake terraform.tfstate at the path deploy.LocalTfStatePath
// will resolve to for workDir + target. Keeps the summary tests self-contained
// without plumbing test helpers into production code.
func seedTfstate(t *testing.T, workDir, target, body string) {
	t.Helper()
	dir := filepath.Join(workDir, ".databricks", "ucm", target, "terraform")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "terraform.tfstate"), []byte(body), 0o600))
}

// writeUcmYml drops a ucm.yml file in a fresh temp dir and returns the dir so
// the summary tests can drive runVerbInDir against an in-line fixture without
// cloning the valid/ testdata tree.
func writeUcmYml(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "ucm.yml"), []byte(body), 0o600))
	return dir
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

func TestCmd_Summary_ListsCatalogsAndSchemasWhenDeployed(t *testing.T) {
	work := cloneFixture(t, validFixtureDir(t))
	seedTfstate(t, work, "default", `{
  "version": 4,
  "resources": [
    {"type": "databricks_catalog", "name": "team_alpha", "mode": "managed", "instances": [{"attributes": {"id": "team_alpha"}}]},
    {"type": "databricks_schema",  "name": "bronze",     "mode": "managed", "instances": [{"attributes": {"id": "team_alpha.bronze"}}]}
  ]
}`)

	stdout, _, err := runVerbInDir(t, work, "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Catalogs:")
	assert.Contains(t, stdout, "team_alpha:")
	assert.Contains(t, stdout, "Name: team_alpha")
	assert.Contains(t, stdout, "explore/data/team_alpha")
	assert.Contains(t, stdout, "Schemas:")
	assert.Contains(t, stdout, "bronze:")
	assert.Contains(t, stdout, "Name: team_alpha.bronze")
	assert.Contains(t, stdout, "explore/data/team_alpha/bronze")
}

// TestCmd_Summary_ListsCatalogsAndSchemasWhenNotDeployed is the DAB-parity
// case that prompted this fix: no local tfstate exists, so every URL-bearing
// resource must render "(not deployed)" instead of a URL that 404s.
func TestCmd_Summary_ListsCatalogsAndSchemasWhenNotDeployed(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Catalogs:")
	assert.Contains(t, stdout, "team_alpha:")
	assert.Contains(t, stdout, "URL:  (not deployed)")
	assert.Contains(t, stdout, "Schemas:")
	assert.Contains(t, stdout, "bronze:")
	// No workspace-console URL should appear anywhere in the output.
	assert.NotContains(t, stdout, "explore/data/team_alpha")
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

func TestCmd_Summary_ListsStorageCredentialsWhenDeployed(t *testing.T) {
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
	seedTfstate(t, work, "default", `{
  "version": 4,
  "resources": [
    {"type": "databricks_storage_credential", "name": "sales_cred", "mode": "managed", "instances": [{"attributes": {"id": "sales_cred"}}]}
  ]
}`)

	stdout, _, err := runVerbInDir(t, work, "summary")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Storage credentials:")
	assert.Contains(t, stdout, "sales_cred:")
	assert.Contains(t, stdout, "Name: sales_cred")
	assert.Contains(t, stdout, "explore/storage-credentials/sales_cred")
}

func TestCmd_Summary_ListsStorageCredentialsWhenNotDeployed(t *testing.T) {
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
	assert.Contains(t, stdout, "URL:  (not deployed)")
	assert.NotContains(t, stdout, "explore/storage-credentials/sales_cred")
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

func TestCmd_Summary_IncludeLocationsOffOmitsLocationsKey(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary", "--output", "json")
	require.NoError(t, err)

	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))
	_, ok := tree["__locations"]
	assert.False(t, ok, "default summary JSON should not contain __locations")
}

func TestCmd_Summary_IncludeLocationsOnAddsLocationsKey(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary", "--output", "json", "--include-locations")
	require.NoError(t, err)

	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))
	locs, ok := tree["__locations"].(map[string]any)
	require.True(t, ok, "expected __locations in JSON output: %s", stdout)
	assert.Contains(t, locs, "files")
	assert.Contains(t, locs, "locations")
}

// TestCmd_Summary_ShowFullConfigDumpsResolvedConfig exercises the
// --show-full-config branch: output is valid JSON and reflects the
// post-mutator-chain config (expanded workspace root, resolved resources).
func TestCmd_Summary_ShowFullConfigDumpsResolvedConfig(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary", "--show-full-config")
	require.NoError(t, err)

	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))

	workspace, ok := tree["workspace"].(map[string]any)
	require.True(t, ok, "expected workspace block in JSON output: %s", stdout)
	// DefineDefaultWorkspaceRoot + ExpandWorkspaceRoot have run.
	rootPath, _ := workspace["root_path"].(string)
	assert.Contains(t, rootPath, "/Workspace/Users/test-user@example.com", "expected expanded workspace root: %q", rootPath)

	resources, ok := tree["resources"].(map[string]any)
	require.True(t, ok, "expected resources block in JSON output")
	catalogs, ok := resources["catalogs"].(map[string]any)
	require.True(t, ok, "expected catalogs in JSON output")
	alpha, ok := catalogs["team_alpha"].(map[string]any)
	require.True(t, ok, "expected team_alpha catalog in JSON output")
	assert.Equal(t, "team_alpha", alpha["name"])
}

// TestCmd_Summary_ShowFullConfigBypassesGroupedText ensures the flag diverts
// away from the grouped text renderer even when --output is not json.
func TestCmd_Summary_ShowFullConfigBypassesGroupedText(t *testing.T) {
	stdout, _, err := runVerb(t, validFixtureDir(t), "summary", "--show-full-config")
	require.NoError(t, err)

	assert.NotContains(t, stdout, "Catalogs:\n")
	assert.NotContains(t, stdout, "Schemas:\n")
	// Valid JSON document, not the text header.
	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))
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
	renderSummaryText(&buf, u)

	out := buf.String()
	assert.Contains(t, out, "Catalogs:")
	assert.NotContains(t, out, "Schemas:")
	assert.NotContains(t, out, "Grants:")
	assert.NotContains(t, out, "Storage credentials:")
}
