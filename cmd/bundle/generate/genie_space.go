package generate

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

const genieSpaceWatchInterval = 1 * time.Second

type genieSpace struct {
	// Lookup flags for one-time generate.
	existingPath string
	existingID   string

	// Lookup flag for existing bundle resource.
	resource string

	// Where to write the configuration and genie space representation.
	resourceDir   string
	genieSpaceDir string

	// Force overwrite of existing files.
	force bool

	// Watch for changes to the genie space.
	watch bool

	// Relative path from the resource directory to the genie space directory.
	relativeGenieSpaceDir string

	// Command.
	cmd *cobra.Command

	// Automatically bind the generated resource to the existing resource.
	bind bool

	// Output and error streams.
	out io.Writer
	err io.Writer
}

func (g *genieSpace) resolveID(ctx context.Context, b *bundle.Bundle) string {
	switch {
	case g.existingPath != "":
		return g.resolveFromPath(ctx, b)
	case g.existingID != "":
		return g.resolveFromID(ctx, b)
	}

	logdiag.LogError(ctx, errors.New("expected one of --existing-path, --existing-id"))
	return ""
}

func (g *genieSpace) resolveFromPath(ctx context.Context, b *bundle.Bundle) string {
	w := b.WorkspaceClient(ctx)
	obj, err := w.Workspace.GetStatusByPath(ctx, g.existingPath) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
	if err != nil {
		if apierr.IsMissing(err) {
			logdiag.LogError(ctx, fmt.Errorf("genie space %q not found", path.Base(g.existingPath)))
			return ""
		}

		logdiag.LogError(ctx, err)
		return ""
	}

	if obj.ResourceId == "" {
		logdiag.LogDiag(ctx, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "expected a non-empty genie space resource ID",
		})
		return ""
	}

	return obj.ResourceId
}

func (g *genieSpace) resolveFromID(ctx context.Context, b *bundle.Bundle) string {
	w := b.WorkspaceClient(ctx)
	obj, err := w.Genie.GetSpace(ctx, dashboards.GenieGetSpaceRequest{
		SpaceId: g.existingID,
	})
	if err != nil {
		// The Genie API returns 403 (not 404) when a space does not exist.
		if apierr.IsMissing(err) || errors.Is(err, apierr.ErrPermissionDenied) {
			logdiag.LogError(ctx, fmt.Errorf("genie space with ID %s not found", g.existingID))
			return ""
		}
		logdiag.LogError(ctx, err)
		return ""
	}

	return obj.SpaceId
}

func (g *genieSpace) saveSerializedGenieSpace(ctx context.Context, b *bundle.Bundle, genieSpace *dashboards.GenieSpace, filename string) error {
	if genieSpace.SerializedSpace == "" {
		return fmt.Errorf("genie space response did not include serialized_space; refusing to write %s", filepath.ToSlash(filename))
	}

	// Unmarshal and remarshal the serialized genie space to ensure it is formatted correctly.
	// The result will have alphabetically sorted keys and be indented.
	data, err := remarshalJSON([]byte(genieSpace.SerializedSpace))
	if err != nil {
		return err
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}

	// Clean the filename to ensure it is a valid path (and can be used on this OS).
	filename = filepath.Clean(filename)

	// Attempt to make the path relative to the bundle root.
	rel, err := filepath.Rel(b.BundleRootPath, filename)
	if err != nil {
		rel = filename
	}

	// Verify that the file does not already exist.
	info, err := os.Stat(filename)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", filepath.ToSlash(rel))
		}
		if !g.force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", filepath.ToSlash(rel))
		}
	}

	cmdio.LogString(ctx, "Writing genie space to "+filepath.ToSlash(rel))
	return os.WriteFile(filename, data, 0o644)
}

func (g *genieSpace) saveConfiguration(ctx context.Context, b *bundle.Bundle, genieSpace *dashboards.GenieSpace, key string) error {
	// Save serialized genie space definition to the genie space directory.
	genieSpaceBasename := key + ".geniespace.json"
	genieSpacePath := filepath.Join(g.genieSpaceDir, genieSpaceBasename)
	err := g.saveSerializedGenieSpace(ctx, b, genieSpace, genieSpacePath)
	if err != nil {
		return err
	}

	// Synthesize resource configuration.
	v, err := generate.ConvertGenieSpaceToValue(genieSpace, path.Join(g.relativeGenieSpaceDir, genieSpaceBasename))
	if err != nil {
		return err
	}

	result := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			"genie_spaces": dyn.V(map[string]dyn.Value{
				key: v,
			}),
		}),
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(g.resourceDir, 0o755); err != nil {
		return err
	}

	// Save the configuration to the resource directory.
	resourcePath := filepath.Join(g.resourceDir, key+".genie_space.yml")
	saver := yamlsaver.NewSaverWithStyle(map[string]yaml.Style{
		"title": yaml.DoubleQuotedStyle,
	})

	// Attempt to make the path relative to the bundle root.
	rel, err := filepath.Rel(b.BundleRootPath, resourcePath)
	if err != nil {
		rel = resourcePath
	}

	cmdio.LogString(ctx, "Writing configuration to "+filepath.ToSlash(rel))
	err = saver.SaveAsYAML(result, resourcePath, g.force)
	if err != nil {
		return err
	}

	return nil
}

func (g *genieSpace) updateGenieSpaceForResource(ctx context.Context, b *bundle.Bundle) {
	resource, ok := b.Config.Resources.GenieSpaces[g.resource]
	if !ok {
		logdiag.LogError(ctx, fmt.Errorf("genie space resource %q is not defined", g.resource))
		return
	}

	if resource.FilePath == "" {
		logdiag.LogError(ctx, fmt.Errorf("genie space resource %q has no file path defined", g.resource))
		return
	}

	genieSpaceID := resource.ID
	genieSpacePath := resource.FilePath

	w := b.WorkspaceClient(ctx)

	first := true
	for {
		genieSpace, err := w.Genie.GetSpace(ctx, dashboards.GenieGetSpaceRequest{
			SpaceId:                genieSpaceID,
			IncludeSerializedSpace: true,
		})
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}

		// Genie has no remote modification timestamp we can poll. Compare
		// the canonicalized remote body against the on-disk body and only
		// re-save when they differ. The first iteration always saves, to
		// match the prior behavior of an unconditional initial sync.
		shouldSave := first
		if !first {
			differs, err := genieSpaceBodyDiffersFromDisk(genieSpace.SerializedSpace, genieSpacePath)
			if err != nil {
				logdiag.LogError(ctx, err)
				return
			}
			shouldSave = differs
		}

		if shouldSave {
			if err := g.saveSerializedGenieSpace(ctx, b, genieSpace, genieSpacePath); err != nil {
				logdiag.LogError(ctx, err)
				return
			}
		}

		if !g.watch {
			return
		}

		first = false
		select {
		case <-ctx.Done():
			return
		case <-time.After(genieSpaceWatchInterval):
		}
	}
}

// genieSpaceBodyDiffersFromDisk reports whether the canonicalized remote
// serialized_space differs from the contents of filename.
func genieSpaceBodyDiffersFromDisk(remoteSerialized, filename string) (bool, error) {
	if remoteSerialized == "" {
		return false, nil
	}
	canonical, err := remarshalJSON([]byte(remoteSerialized))
	if err != nil {
		return false, err
	}
	onDisk, err := os.ReadFile(filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}
	return !bytes.Equal(canonical, onDisk), nil
}

func (g *genieSpace) generateForExisting(ctx context.Context, b *bundle.Bundle, genieSpaceID string) {
	w := b.WorkspaceClient(ctx)
	genieSpace, err := w.Genie.GetSpace(ctx, dashboards.GenieGetSpaceRequest{
		SpaceId:                genieSpaceID,
		IncludeSerializedSpace: true,
	})
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	key := g.cmd.Flag("key").Value.String()
	if key == "" {
		key = textutil.NormalizeString(genieSpace.Title)
	}
	err = g.saveConfiguration(ctx, b, genieSpace, key)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	if g.bind {
		err = deployment.BindResource(g.cmd, key, genieSpaceID, true, false, true)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		cmdio.LogString(ctx, fmt.Sprintf("Successfully bound genie space with an id '%s'", genieSpaceID))
	}
}

func (g *genieSpace) initialize(ctx context.Context, b *bundle.Bundle) {
	// Make the paths absolute if they aren't already.
	if !filepath.IsAbs(g.resourceDir) {
		g.resourceDir = filepath.Join(b.BundleRootPath, g.resourceDir)
	}
	if !filepath.IsAbs(g.genieSpaceDir) {
		g.genieSpaceDir = filepath.Join(b.BundleRootPath, g.genieSpaceDir)
	}

	// Make sure we know how the genie space path is relative to the resource path.
	rel, err := filepath.Rel(g.resourceDir, g.genieSpaceDir)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	g.relativeGenieSpaceDir = filepath.ToSlash(rel)
}

func (g *genieSpace) runForResource(ctx context.Context, b *bundle.Bundle) {
	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	requiredEngine, err := utils.ResolveEngineSetting(ctx, b)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	ctx, stateDesc := statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(true), requiredEngine)
	if logdiag.HasError(ctx) {
		return
	}

	var state statemgmt.ExportedResourcesMap
	if stateDesc.Engine.IsDirect() {
		_, localPath := b.StateFilenameDirect(ctx)
		if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false)); err != nil {
			logdiag.LogError(ctx, err)
			return
		}
		state = b.DeploymentBundle.ExportState(ctx)
	} else {
		var err error
		state, err = terraform.ParseResourcesState(ctx, b)
		if err != nil {
			logdiag.LogError(ctx, err)
			return
		}
	}

	bundle.ApplySeqContext(ctx, b,
		statemgmt.Load(state),
	)
	if logdiag.HasError(ctx) {
		return
	}

	g.updateGenieSpaceForResource(ctx, b)
}

func (g *genieSpace) runForExisting(ctx context.Context, b *bundle.Bundle) {
	// Resolve the ID of the genie space to generate configuration for.
	genieSpaceID := g.resolveID(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	g.generateForExisting(ctx, b, genieSpaceID)
}

func (g *genieSpace) RunE(cmd *cobra.Command, args []string) error {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	b := root.MustConfigureBundle(cmd)
	if b == nil || logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	g.initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	if g.resource != "" {
		g.runForResource(ctx, b)
	} else {
		g.runForExisting(ctx, b)
	}

	if logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	return nil
}

// filterGenieSpaces returns a filter that only includes genie spaces.
func filterGenieSpaces(ref resources.Reference) bool {
	return ref.Description.SingularName == "genie_space"
}

// genieSpaceResourceCompletion executes to autocomplete the argument to the resource flag.
func genieSpaceResourceCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	b := root.MustConfigureBundle(cmd)
	if logdiag.HasError(cmd.Context()) {
		return nil, cobra.ShellCompDirectiveError
	}

	if b == nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return slices.Collect(maps.Keys(resources.Completions(b, filterGenieSpaces))), cobra.ShellCompDirectiveNoFileComp
}

func NewGenerateGenieSpaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genie-space",
		Short: "Generate configuration for a Genie space",
		Long: `Generate bundle configuration for an existing Databricks Genie space.

This command downloads an existing Genie space and creates bundle files
that you can use to deploy the Genie space to other environments or manage it as code.

Examples:
  # Import Genie space by workspace path
  databricks bundle generate genie-space --existing-path /Users/me/my-genie-space

  # Import Genie space by ID
  databricks bundle generate genie-space --existing-id abc123

  # Watch for changes to keep bundle in sync with UI modifications
  databricks bundle generate genie-space --resource my_genie_space --watch --force

What gets generated:
- Genie space configuration YAML file with settings and a reference to the Genie space definition
- Genie space definition (.geniespace.json) file with the serialized space content

Sync workflow for Genie space development:
When developing Genie spaces, you can modify them in the Databricks UI and sync
changes back to your bundle:

1. Make changes to Genie space in the Databricks UI
2. Run: databricks bundle generate genie-space --resource my_genie_space --force
3. Commit changes to version control
4. Deploy to other environments with: databricks bundle deploy --target prod

The --watch flag continuously polls for remote changes and updates your local
bundle files automatically, useful during active Genie space development.`,
	}

	g := &genieSpace{
		out: cmd.OutOrStdout(),
		err: cmd.ErrOrStderr(),
	}

	// Lookup flags.
	cmd.Flags().StringVar(&g.existingPath, "existing-path", "", `workspace path of the Genie space to generate configuration for`)
	cmd.Flags().StringVar(&g.existingID, "existing-id", "", `ID of the Genie space to generate configuration for`)
	cmd.Flags().StringVar(&g.resource, "resource", "", `resource key of Genie space to watch for changes`)

	// Alias lookup flags that include the resource type name.
	cmd.Flags().StringVar(&g.existingPath, "existing-genie-space-path", "", `workspace path of the Genie space to generate configuration for`)
	cmd.Flags().StringVar(&g.existingID, "existing-genie-space-id", "", `ID of the Genie space to generate configuration for`)
	cmd.Flags().MarkHidden("existing-genie-space-path")
	cmd.Flags().MarkHidden("existing-genie-space-id")

	// Output flags.
	cmd.Flags().StringVarP(&g.resourceDir, "resource-dir", "d", "resources", `directory to write the configuration to`)
	cmd.Flags().StringVarP(&g.genieSpaceDir, "genie-space-dir", "s", "src", `directory to write the Genie space representation to`)
	cmd.Flags().BoolVarP(&g.force, "force", "f", false, `force overwrite existing files in the output directory`)

	cmd.Flags().BoolVarP(&g.bind, "bind", "b", false, `automatically bind the generated Genie space config to the existing Genie space`)
	cmd.Flags().MarkHidden("bind")

	// Exactly one of the lookup flags must be provided.
	cmd.MarkFlagsOneRequired(
		"existing-path",
		"existing-id",
		"resource",
	)

	// Watch flag. This is relevant only in combination with the resource flag.
	cmd.Flags().BoolVar(&g.watch, "watch", false, `watch for changes to the Genie space and update the configuration`)

	// Make sure the watch flag is only used with the existing-resource flag.
	cmd.MarkFlagsMutuallyExclusive("watch", "existing-path")
	cmd.MarkFlagsMutuallyExclusive("watch", "existing-id")

	// Make sure the bind flag is only used with the existing-resource flag.
	cmd.MarkFlagsMutuallyExclusive("bind", "resource")

	// Completion for the resource flag.
	cmd.RegisterFlagCompletionFunc("resource", genieSpaceResourceCompletion)

	cmd.RunE = g.RunE
	g.cmd = cmd
	return cmd
}
