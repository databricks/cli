package bundle_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

const defaultSparkVersion = "13.3.x-snapshot-scala2.12"

func initTestTemplate(t testutil.TestingT, ctx context.Context, templateName string, config map[string]any) string {
	bundleRoot := t.TempDir()
	return initTestTemplateWithBundleRoot(t, ctx, templateName, config, bundleRoot)
}

func initTestTemplateWithBundleRoot(t testutil.TestingT, ctx context.Context, templateName string, config map[string]any, bundleRoot string) string {
	templateRoot := filepath.Join("bundles", templateName)

	configFilePath := writeConfigFile(t, config)

	ctx = root.SetWorkspaceClient(ctx, nil)
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

func validateBundle(t testutil.TestingT, ctx context.Context, path string) ([]byte, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := testcli.NewRunner(t, ctx, "bundle", "validate", "--output", "json")
	stdout, _, err := c.Run()
	return stdout.Bytes(), err
}

func mustValidateBundle(t testutil.TestingT, ctx context.Context, path string) []byte {
	data, err := validateBundle(t, ctx, path)
	require.NoError(t, err)
	return data
}

func unmarshalConfig(t testutil.TestingT, data []byte) *bundle.Bundle {
	bundle := &bundle.Bundle{}
	err := json.Unmarshal(data, &bundle.Config)
	require.NoError(t, err)
	return bundle
}

func deployBundle(t testutil.TestingT, ctx context.Context, path string) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err := c.Run()
	require.NoError(t, err)
}

func deployBundleWithArgsErr(t testutil.TestingT, ctx context.Context, path string, args ...string) (string, string, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	args = append([]string{"bundle", "deploy"}, args...)
	c := testcli.NewRunner(t, ctx, args...)
	stdout, stderr, err := c.Run()
	return stdout.String(), stderr.String(), err
}

func deployBundleWithArgs(t testutil.TestingT, ctx context.Context, path string, args ...string) (string, string) {
	stdout, stderr, err := deployBundleWithArgsErr(t, ctx, path, args...)
	require.NoError(t, err)
	return stdout, stderr
}

func deployBundleWithFlags(t testutil.TestingT, ctx context.Context, path string, flags []string) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	args := []string{"bundle", "deploy", "--force-lock"}
	args = append(args, flags...)
	c := testcli.NewRunner(t, ctx, args...)
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

func runResourceWithStderr(t testutil.TestingT, ctx context.Context, path, key string) (string, string) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	c := testcli.NewRunner(t, ctx, "bundle", "run", key)
	stdout, stderr, err := c.Run()
	require.NoError(t, err)

	return stdout.String(), stderr.String()
}

func runResourceWithParams(t testutil.TestingT, ctx context.Context, path, key string, params ...string) (string, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	args := make([]string, 0)
	args = append(args, "bundle", "run", key)
	args = append(args, params...)
	c := testcli.NewRunner(t, ctx, args...)
	stdout, _, err := c.Run()
	return stdout.String(), err
}

func destroyBundle(t testutil.TestingT, ctx context.Context, path string) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := testcli.NewRunner(t, ctx, "bundle", "destroy", "--auto-approve")
	_, _, err := c.Run()
	require.NoError(t, err)
}

func getBundleRemoteRootPath(w *databricks.WorkspaceClient, t testutil.TestingT, uniqueId string) string {
	// Compute root path for the bundle deployment
	me, err := w.CurrentUser.Me(context.Background())
	require.NoError(t, err)
	root := fmt.Sprintf("/Workspace/Users/%s/.bundle/%s", me.UserName, uniqueId)
	return root
}

func blackBoxRun(t testutil.TestingT, ctx context.Context, root string, args ...string) (stdout, stderr string) {
	gitRoot, err := folders.FindDirWithLeaf(".", ".git")
	require.NoError(t, err)

	// Create the command
	cmd := exec.Command("go", append([]string{"run", "main.go"}, args...)...)
	cmd.Dir = gitRoot

	// Configure the environment
	ctx = env.Set(ctx, "BUNDLE_ROOT", root)
	for key, value := range env.All(ctx) {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create buffers to capture output
	var outBuffer, errBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run the command
	err = cmd.Run()
	require.NoError(t, err)

	// Get the output
	stdout = outBuffer.String()
	stderr = errBuffer.String()
	return
}
