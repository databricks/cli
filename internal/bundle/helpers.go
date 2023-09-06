package bundle

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/template"
)

//go:embed all:bundles
var builtinBundles embed.FS

func copyBuiltInTemplate(templateName string, dst string) (string, error) {
	_, err := fs.Stat(builtinBundles, path.Join("bundles", templateName))
	if err != nil {
		return "", err
	}

	err = fs.WalkDir(builtinBundles, "bundles", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(dst, path)
		if entry.IsDir() {
			return os.Mkdir(targetPath, 0755)
		} else {
			content, err := fs.ReadFile(builtinBundles, path)
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

	bundelRoot := t.TempDir()
	configFilePath, err := writeConfigFile(t, config)
	if err != nil {
		return "", err
	}

	ctx := root.SetWorkspaceClient(context.Background(), nil)
	cmd := cmdio.NewIO(flags.OutputJSON, strings.NewReader(""), os.Stdout, os.Stderr, "bundles")
	ctx = cmdio.InContext(ctx, cmd)

	err = template.Materialize(ctx, configFilePath, templateRoot, bundelRoot)
	return bundelRoot, err
}

func writeConfigFile(t *testing.T, config map[string]any) (string, error) {
	bytes, err := json.Marshal(config)
	if err != nil {
		return "", err
	}

	dir := t.TempDir()
	filepath := filepath.Join(dir, "config.json")
	fmt.Println(string(bytes))
	err = os.WriteFile(filepath, bytes, 0644)
	return filepath, err
}

func loadBundle(ctx context.Context, path string) (*bundle.Bundle, error) {
	b, err := bundle.Load(ctx, path)
	if err != nil {
		return nil, err
	}

	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	if err != nil {
		return nil, err
	}

	err = bundle.Apply(ctx, b, mutator.SelectDefaultTarget())
	if err != nil {
		return nil, err
	}

	return b, err
}

func deployBundle(path string) (*bundle.Bundle, error) {
	ctx := context.Background()
	b, err := loadBundle(ctx, path)
	if err != nil {
		return nil, err
	}

	err = bundle.Apply(ctx, b, phases.Initialize())
	if err != nil {
		return nil, err
	}

	err = bundle.Apply(ctx, b, phases.Build())
	if err != nil {
		return nil, err
	}

	b.AutoApprove = true
	b.Config.Bundle.Lock.Force = true

	err = bundle.Apply(ctx, b, phases.Deploy())
	return b, err
}

func runResource(path string, key string) (string, error) {
	ctx := context.Background()
	ctx = cmdio.NewContext(ctx, cmdio.Default())
	b, err := loadBundle(ctx, path)
	if err != nil {
		return "", err
	}

	err = bundle.Apply(ctx, b, bundle.Seq(
		phases.Initialize(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.StatePull(),
		terraform.Load(),
	))

	if err != nil {
		return "", err
	}

	runner, err := run.Find(b, "some_other_job")
	if err != nil {
		return "", err
	}

	output, err := runner.Run(ctx, &run.Options{})
	if err != nil {
		return "", err
	}

	return output.String()
}

func destroyBundle(b *bundle.Bundle) error {
	ctx := context.Background()
	return bundle.Apply(ctx, b, phases.Destroy())
}
