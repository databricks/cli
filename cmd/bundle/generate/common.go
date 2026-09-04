package generate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/phases"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

func configureBundle(cmd *cobra.Command) (context.Context, *bundle.Bundle, error) {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	b := root.MustConfigureBundle(cmd)
	if b == nil || logdiag.HasError(ctx) {
		return ctx, nil, root.ErrAlreadyPrinted
	}

	return ctx, b, nil
}

func errAlreadyPrinted() error {
	return root.ErrAlreadyPrinted
}

func addOutputDirFlag(cmd *cobra.Command, target *string, name, shorthand, defaultValue, usage string) {
	cmd.Flags().StringVarP(target, name, shorthand, defaultValue, usage)
}

func addForceFlag(cmd *cobra.Command, target *bool, usage string) {
	cmd.Flags().BoolVarP(target, "force", "f", false, usage)
}

func addHiddenBindFlag(cmd *cobra.Command, target *bool, usage string) {
	cmd.Flags().BoolVarP(target, "bind", "b", false, usage)
	_ = cmd.Flags().MarkHidden("bind")
}

func selectedResourceKey(cmd *cobra.Command, defaultName string) string {
	key := cmd.Flag("key").Value.String()
	if key == "" {
		key = textutil.NormalizeString(defaultName)
	}
	return key
}

func generatedResourceConfig(resourceType, key string, value dyn.Value) map[string]dyn.Value {
	return map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			resourceType: dyn.V(map[string]dyn.Value{
				key: value,
			}),
		}),
	}
}

func saveGeneratedResourceConfig(result map[string]dyn.Value, filename string, force bool, styles map[string]yaml.Style) error {
	saver := yamlsaver.NewSaver()
	if styles != nil {
		saver = yamlsaver.NewSaverWithStyle(styles)
	}

	return saver.SaveAsYAML(result, filename, force)
}

func migrateGeneratedResourceFilename(configDir, key, resourceType string) (string, error) {
	oldFilename := filepath.Join(configDir, key+".yml")
	filename := filepath.Join(configDir, key+"."+resourceType+".yml")

	err := os.Rename(oldFilename, filename)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("failed to rename file %s. DABs uses the resource type as a sub-extension for generated content, please rename it to %s, err: %w", oldFilename, filename, err)
	}

	return filename, nil
}

func bindGeneratedResource(cmd *cobra.Command, resourceKey, resourceID string) error {
	return deployment.BindResource(cmd, resourceKey, resourceID, true, false, true)
}

func absolutePath(rootPath, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(rootPath, path)
}

func loadBundleResourceState(ctx context.Context, b *bundle.Bundle) context.Context {
	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return ctx
	}

	requiredEngine, err := utils.ResolveEngineSetting(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return ctx
	}
	ctx, stateDesc := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true), requiredEngine)
	if logdiag.HasError(ctx) {
		return ctx
	}

	var state statemgmt.ExportedResourcesMap
	if stateDesc.Engine.IsDirect() {
		_, localPath := b.StateFilenameDirect(ctx)
		if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false)); err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}
		state = b.DeploymentBundle.ExportState(ctx)
	} else {
		var err error
		state, err = terraform.ParseResourcesState(ctx, b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return ctx
		}
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(state),
	)
	return ctx
}

func resourceKeyCompletion(filter func(bundleresources.Reference) bool) func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		b := root.MustConfigureBundle(cmd)
		if logdiag.HasError(cmd.Context()) {
			return nil, cobra.ShellCompDirectiveError
		}

		if b == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		return slices.Collect(maps.Keys(bundleresources.Completions(b, filter))), cobra.ShellCompDirectiveNoFileComp
	}
}
