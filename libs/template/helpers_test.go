package template

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatePrintStringWithoutProcessing(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ctx = root.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/print-without-processing/template", "./testdata/print-without-processing/library", tmpDir)
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	cleanContent := strings.Trim(string(r.files[0].(*inMemoryFile).content), "\n\r")
	assert.Equal(t, `{{ fail "abc" }}`, cleanContent)
}

func TestTemplateRegexpCompileFunction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ctx = root.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/regexp-compile/template", "./testdata/regexp-compile/library", tmpDir)
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	content := string(r.files[0].(*inMemoryFile).content)
	assert.Contains(t, content, "0:food")
	assert.Contains(t, content, "1:fool")
}

func TestTemplateUrlFunction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ctx = root.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/urlparse-function/template", "./testdata/urlparse-function/library", tmpDir)

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Equal(t, "https://www.databricks.com", string(r.files[0].(*inMemoryFile).content))
}

func TestTemplateMapPairFunction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	ctx = root.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/map-pair/template", "./testdata/map-pair/library", tmpDir)

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Equal(t, "false 123 hello 12.3", string(r.files[0].(*inMemoryFile).content))
}

func TestWorkspaceHost(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{
			Host: "https://myhost.com",
		},
	}
	ctx = root.SetWorkspaceClient(ctx, w)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/workspace-host/template", "./testdata/map-pair/library", tmpDir)

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Contains(t, string(r.files[0].(*inMemoryFile).content), "https://myhost.com")
	assert.Contains(t, string(r.files[0].(*inMemoryFile).content), "i3.xlarge")

}

func TestWorkspaceHostNotConfigured(t *testing.T) {
	ctx := context.Background()
	cmd := cmdio.NewIO(flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "", "template")
	ctx = cmdio.InContext(ctx, cmd)
	tmpDir := t.TempDir()

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{},
	}
	ctx = root.SetWorkspaceClient(ctx, w)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, "./testdata/workspace-host/template", "./testdata/map-pair/library", tmpDir)

	assert.NoError(t, err)

	err = r.walk()
	require.ErrorContains(t, err, "cannot determine target workspace")

}
