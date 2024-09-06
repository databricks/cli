package bundle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

const defaultSparkVersion = "13.3.x-snapshot-scala2.12"

func initTestTemplate(t *testing.T, ctx context.Context, templateName string, config map[string]any) (string, error) {
	bundleRoot := t.TempDir()
	return initTestTemplateWithBundleRoot(t, ctx, templateName, config, bundleRoot)
}

func initTestTemplateWithBundleRoot(t *testing.T, ctx context.Context, templateName string, config map[string]any, bundleRoot string) (string, error) {
	templateRoot := filepath.Join("bundles", templateName)

	configFilePath, err := writeConfigFile(t, config)
	if err != nil {
		return "", err
	}

	ctx = root.SetWorkspaceClient(ctx, nil)
	cmd := cmdio.NewIO(flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "", "bundles")
	ctx = cmdio.InContext(ctx, cmd)

	err = template.Materialize(ctx, configFilePath, templateRoot, bundleRoot)
	return bundleRoot, err
}

func writeConfigFile(t *testing.T, config map[string]any) (string, error) {
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	dir := t.TempDir()
	filepath := filepath.Join(dir, "config.json")
	t.Log("Configuration for template: ", string(bytes))

	err = os.WriteFile(filepath, bytes, 0644)
	return filepath, err
}

func validateBundle(t *testing.T, ctx context.Context, path string) ([]byte, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "validate", "--output", "json")
	stdout, _, err := c.Run()
	return stdout.Bytes(), err
}

func deployBundle(t *testing.T, ctx context.Context, path string) error {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "deploy", "--force-lock", "--auto-approve")
	_, _, err := c.Run()
	return err
}

func deployBundleWithFlags(t *testing.T, ctx context.Context, path string, flags []string) error {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	args := []string{"bundle", "deploy", "--force-lock"}
	args = append(args, flags...)
	c := internal.NewCobraTestRunnerWithContext(t, ctx, args...)
	_, _, err := c.Run()
	return err
}

func runResource(t *testing.T, ctx context.Context, path string, key string) (string, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "run", key)
	stdout, _, err := c.Run()
	return stdout.String(), err
}

func runResourceWithParams(t *testing.T, ctx context.Context, path string, key string, params ...string) (string, error) {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	args := make([]string, 0)
	args = append(args, "bundle", "run", key)
	args = append(args, params...)
	c := internal.NewCobraTestRunnerWithContext(t, ctx, args...)
	stdout, _, err := c.Run()
	return stdout.String(), err
}

func destroyBundle(t *testing.T, ctx context.Context, path string) error {
	ctx = env.Set(ctx, "BUNDLE_ROOT", path)
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "destroy", "--auto-approve")
	_, _, err := c.Run()
	return err
}

func getBundleRemoteRootPath(w *databricks.WorkspaceClient, t *testing.T, uniqueId string) string {
	// Compute root path for the bundle deployment
	me, err := w.CurrentUser.Me(context.Background())
	require.NoError(t, err)
	root := fmt.Sprintf("/Users/%s/.bundle/%s", me.UserName, uniqueId)
	return root
}

func blackBoxRun(t *testing.T, root string, args ...string) (stdout string, stderr string) {
	cwd := vfs.MustNew(".")
	gitRoot, err := vfs.FindLeafInTree(cwd, ".git")
	require.NoError(t, err)

	t.Setenv("BUNDLE_ROOT", root)

	// Create the command
	cmd := exec.Command("go", append([]string{"run", "main.go"}, args...)...)
	cmd.Dir = gitRoot.Native()

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
