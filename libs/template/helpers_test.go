package template

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	workspaceConfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatePrintStringWithoutProcessing(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/print-without-processing/template", "./testdata/print-without-processing/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	cleanContent := strings.Trim(string(r.files[0].(*inMemoryFile).content), "\n\r")
	assert.Equal(t, `{{ fail "abc" }}`, cleanContent)
}

func TestTemplateBundleUuidFunction(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/bundle-uuid/template", "./testdata/bundle-uuid/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	// We test the content of two generated files to ensure that the same UUID
	// is used across all files generated by the template. Each file content is
	// generated by a separate (*template.Template).Execute call, so testing different
	// files ensures that the UUID is stable across all template execution calls.
	copy := strings.Clone(bundleUuid)
	assert.Len(t, r.files, 2)
	c1 := strings.Trim(string(r.files[0].(*inMemoryFile).content), "\n\r")
	assert.Equal(t, strings.Repeat(copy, 3), c1)
	c2 := strings.Trim(string(r.files[1].(*inMemoryFile).content), "\n\r")
	assert.Equal(t, strings.Repeat(copy, 5), c2)
}

func TestTemplateRegexpCompileFunction(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/regexp-compile/template", "./testdata/regexp-compile/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	content := string(r.files[0].(*inMemoryFile).content)
	assert.Contains(t, content, "0:food")
	assert.Contains(t, content, "1:fool")
}

func TestTemplateRandIntFunction(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/random-int/template", "./testdata/random-int/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	randInt, err := strconv.Atoi(strings.TrimSpace(string(r.files[0].(*inMemoryFile).content)))
	assert.Less(t, randInt, 10)
	assert.NoError(t, err)
}

func TestTemplateUuidFunction(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/uuid/template", "./testdata/uuid/library")
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	uuid := strings.TrimSpace(string(r.files[0].(*inMemoryFile).content))
	assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", uuid)
}

func TestTemplateUrlFunction(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/urlparse-function/template", "./testdata/urlparse-function/library")

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Equal(t, "https://www.databricks.com", string(r.files[0].(*inMemoryFile).content))
}

func TestTemplateMapPairFunction(t *testing.T) {
	ctx := context.Background()

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/map-pair/template", "./testdata/map-pair/library")

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Equal(t, "false 123 hello 12.3", string(r.files[0].(*inMemoryFile).content))
}

func TestWorkspaceHost(t *testing.T) {
	ctx := context.Background()

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{
			Host: "https://myhost.com",
		},
	}
	ctx = cmdctx.SetWorkspaceClient(ctx, w)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/workspace-host/template", "./testdata/map-pair/library")

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	assert.Contains(t, string(r.files[0].(*inMemoryFile).content), "https://myhost.com")
	assert.Contains(t, string(r.files[0].(*inMemoryFile).content), "i3.xlarge")
}

func TestWorkspaceHostNotConfigured(t *testing.T) {
	ctx := context.Background()
	cmd := cmdio.NewIO(ctx, flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "", "template")
	ctx = cmdio.InContext(ctx, cmd)

	w := &databricks.WorkspaceClient{
		Config: &workspaceConfig.Config{},
	}
	ctx = cmdctx.SetWorkspaceClient(ctx, w)

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, nil, helpers, os.DirFS("."), "./testdata/workspace-host/template", "./testdata/map-pair/library")

	assert.NoError(t, err)

	err = r.walk()
	require.ErrorContains(t, err, "cannot determine target workspace")
}
