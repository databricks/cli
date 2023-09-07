package bundle

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/template"
)

func copyBuiltInTemplate(templateName string, dst string) (string, error) {
	filename := filepath.Join("bundles", templateName)
	_, err := os.Stat(filename)
	if err != nil {
		return "", err
	}

	err = filepath.WalkDir("bundles", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, path)
		if entry.IsDir() {
			return os.Mkdir(targetPath, 0755)
		} else {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, content, 0644)
		}
	})

	if err != nil {
		return "", err
	}

	return filepath.Join(dst, "bundles", templateName), nil
}

func initTestTemplate(t *testing.T, templateName string, config map[string]any) (string, error) {
	templateRoot, err := copyBuiltInTemplate(templateName, t.TempDir())
	if err != nil {
		return "", err
	}

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
	ctx := context.Background()
	t.Setenv("BUNDLE_ROOT", path)

	deploy := cmd.New()
	deploy.SetArgs([]string{"bundle", "deploy", "--force-lock"})

	return deploy.ExecuteContext(ctx)
}

func runResource(t *testing.T, path string, key string) (string, error) {
	ctx := context.Background()
	ctx = cmdio.NewContext(ctx, cmdio.Default())

	run := cmd.New()
	run.SetArgs([]string{"bundle", "run", key})
	buffer := new(bytes.Buffer)
	run.SetOut(buffer)

	return buffer.String(), run.ExecuteContext(ctx)
}

func destroyBundle(t *testing.T, path string) error {
	ctx := context.Background()
	t.Setenv("BUNDLE_ROOT", path)

	deploy := cmd.New()
	deploy.SetArgs([]string{"bundle", "destroy", "--auto-approve"})

	return deploy.ExecuteContext(ctx)
}
