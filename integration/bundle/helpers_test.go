package bundle_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/template"
	"github.com/stretchr/testify/require"
)

func initTestTemplate(t testutil.TestingT, ctx context.Context, templateName string, config map[string]any) string {
	bundleRoot := t.TempDir()
	return initTestTemplateWithBundleRoot(t, ctx, templateName, config, bundleRoot)
}

func initTestTemplateWithBundleRoot(t testutil.TestingT, ctx context.Context, templateName string, config map[string]any, bundleRoot string) string {
	templateRoot := filepath.Join("bundles", templateName)

	configFilePath := writeConfigFile(t, config)

	ctx = cmdctx.SetWorkspaceClient(ctx, nil)
	cmd := cmdio.NewIO(ctx, flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "", "bundles")
	ctx = cmdio.InContext(ctx, cmd)

	r := template.Resolver{
		TemplatePathOrUrl: templateRoot,
		ConfigFile:        configFilePath,
		OutputDir:         bundleRoot,
	}

	tmpl, err := r.Resolve(ctx)
	require.NoError(t, err)
	defer tmpl.Reader.Cleanup(ctx)

	err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
	require.NoError(t, err)

	return bundleRoot
}

func writeConfigFile(t testutil.TestingT, config map[string]any) string {
	bytes, err := json.Marshal(config)
	require.NoError(t, err)

	dir := t.TempDir()
	filepath := filepath.Join(dir, "config.json")
	t.Log("Configuration for template: ", string(bytes))

	err = os.WriteFile(filepath, bytes, 0o644)
	require.NoError(t, err)
	return filepath
}

func deployBundle(t testutil.TestingT, ctx context.Context, path string) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err := c.Run()
	require.NoError(t, err)
}

func runResource(t testutil.TestingT, ctx context.Context, path, key string) (string, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	c := testcli.NewRunner(t, ctx, "bundle", "run", key)
	stdout, _, err := c.Run()
	return stdout.String(), err
}

func destroyBundle(t testutil.TestingT, ctx context.Context, path string) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := testcli.NewRunner(t, ctx, "bundle", "destroy", "--auto-approve")
	_, _, err := c.Run()
	require.NoError(t, err)
}
