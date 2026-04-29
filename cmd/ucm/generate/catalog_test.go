package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// stripPreRun clears auth-hook PreRunE/PersistentPreRunE recursively. The
// real generate subcommands set root.MustWorkspaceClient as PreRunE; tests
// stand in their own mock workspace client via cmdctx so the real hook
// would just fail on a missing ~/.databrickscfg.
func stripPreRun(cmd *cobra.Command) {
	cmd.PersistentPreRunE = nil
	cmd.PersistentPreRun = nil
	cmd.PreRunE = nil
	cmd.PreRun = nil
	for _, sub := range cmd.Commands() {
		stripPreRun(sub)
	}
}

// runSubcmd executes the named subcommand from the parent generate group
// with cmdctx pre-populated with the mock workspace client. Returns stderr
// (captured cmdio output) and the cobra error. Args are passed verbatim;
// callers include both the subcommand name and any flags.
func runSubcmd(t *testing.T, w *mocks.MockWorkspaceClient, args ...string) (string, error) {
	t.Helper()
	cmd := New()
	stripPreRun(cmd)

	ctx := context.Background()
	var stderr bytes.Buffer
	cmdIO := cmdio.NewIO(ctx, flags.OutputText, nil, &bytes.Buffer{}, &stderr, "", "")
	ctx = cmdio.InContext(ctx, cmdIO)
	ctx = cmdctx.SetWorkspaceClient(ctx, w.WorkspaceClient)
	cmd.SetContext(ctx)

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stderr.String(), err
}

func TestNewGenerateCatalogCommand_Help(t *testing.T) {
	cmd := NewGenerateCatalogCommand()
	assert.Contains(t, cmd.Long, "existing Unity Catalog catalog")
	assert.NotNil(t, cmd.Flag("existing-catalog-name"))
	assert.NotNil(t, cmd.Flag("output-dir"))
	assert.NotNil(t, cmd.Flag("force"))
}

func TestGenerateCatalog_WritesYAML(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "team_alpha").
		Return(&catalog.CatalogInfo{
			Name:       "team_alpha",
			Comment:    "alpha team",
			Properties: map[string]string{"owner": "alpha"},
		}, nil)

	stderr, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "team_alpha",
		"--output-dir", work,
	)
	require.NoError(t, err, "stderr=%s", stderr)

	out := filepath.Join(work, "catalogs_team_alpha.yml")
	data, err := os.ReadFile(out)
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "team_alpha")
	assert.Contains(t, contents, "alpha team")
	assert.Contains(t, contents, "catalogs:")
	assert.Contains(t, stderr, "Wrote catalog")
}

func TestGenerateCatalog_KeyOverride(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "raw").
		Return(&catalog.CatalogInfo{Name: "raw"}, nil)

	_, err := runSubcmd(t, w,
		"--key", "raw_alias",
		"catalog",
		"--existing-catalog-name", "raw",
		"--output-dir", work,
	)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(work, "catalogs_raw_alias.yml"))
	require.NoError(t, err)
}

func TestGenerateCatalog_RefusesOverwriteWithoutForce(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "catalogs_x.yml"), []byte("old"), 0o644))

	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "x").
		Return(&catalog.CatalogInfo{Name: "x"}, nil)

	_, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "x",
		"--output-dir", work,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestGenerateCatalog_ForceOverwrites(t *testing.T) {
	work := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(work, "catalogs_x.yml"), []byte("old"), 0o644))

	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockCatalogsAPI().EXPECT().
		GetByName(mock.Anything, "x").
		Return(&catalog.CatalogInfo{Name: "x", Comment: "fresh"}, nil)

	_, err := runSubcmd(t, w,
		"catalog",
		"--existing-catalog-name", "x",
		"--output-dir", work,
		"--force",
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "catalogs_x.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "fresh")
	assert.NotContains(t, string(data), "old")
}
