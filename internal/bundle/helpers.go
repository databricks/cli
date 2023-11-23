package bundle

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/template"
)

func initTestTemplate(t *testing.T, templateName string, config map[string]any) (string, error) {
	templateRoot := filepath.Join("bundles", templateName)

	bundleRoot := t.TempDir()
	configFilePath, err := writeConfigFile(t, config)
	if err != nil {
		return "", err
	}

	ctx := root.SetWorkspaceClient(context.Background(), nil)
	cmd := cmdio.NewIO(flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "bundles")
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

func deployBundle(t *testing.T, path string) error {
	return deployBundleWithVariables(t, path, nil)
}

func deployBundleWithVariables(t *testing.T, path string, vars map[string]string) error {
	t.Setenv("BUNDLE_ROOT", path)
	args := append([]string{"bundle", "deploy", "--force-lock"}, buildVariableArgs(vars)...)
	c := internal.NewCobraTestRunner(t, args...)
	_, _, err := c.Run()
	return err
}

func buildVariableArgs(vars map[string]string) []string {
	var args []string
	for k, v := range vars {
		args = append(args, "--var", k+"="+v)
	}
	return args
}

func runResource(t *testing.T, path string, key string) (string, error) {
	ctx := context.Background()
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "run", key)
	stdout, _, err := c.Run()
	return stdout.String(), err
}

func destroyBundle(t *testing.T, path string) error {
	t.Setenv("BUNDLE_ROOT", path)
	c := internal.NewCobraTestRunner(t, "bundle", "destroy", "--auto-approve")
	_, _, err := c.Run()
	return err
}
