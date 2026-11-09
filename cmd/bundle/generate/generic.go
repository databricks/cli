package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/cmd/bundle/deployment"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/textutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

// noGenericCommand lists the resource groups the generic command skips: the
// hand-written generators also download files, job runs are triggered rather than
// imported, and snapshots are internal.
var noGenericCommand = map[string]bool{
	"jobs":                         true,
	"pipelines":                    true,
	"dashboards":                   true,
	"alerts":                       true,
	"apps":                         true,
	"genie_spaces":                 true,
	"job_runs":                     true,
	"internal_immutable_snapshots": true,
}

// NewResourceCommands returns a generate subcommand for every resource type the
// deployment engine can read, except those in noGenericCommand.
func NewResourceCommands() []*cobra.Command {
	descriptions := config.SupportedResources()
	var cmds []*cobra.Command
	for _, group := range slices.Sorted(maps.Keys(dresources.SupportedResources)) {
		// Permissions and grants ("jobs.permissions") are not standalone resources.
		if noGenericCommand[group] || strings.Contains(group, ".") {
			continue
		}
		cmds = append(cmds, newResourceCommand(group, descriptions[group]))
	}
	return cmds
}

// resourceOptions carries the flag values of a generic generate subcommand.
type resourceOptions struct {
	id        string
	key       string
	configDir string
	force     bool
}

func newResourceCommand(group string, desc resources.ResourceDescription) *cobra.Command {
	var opts resourceOptions
	var bind bool

	use := strings.ReplaceAll(desc.SingularName, "_", "-")
	title := strings.ToLower(desc.SingularTitle)
	cmd := &cobra.Command{
		Use:   use,
		Short: "Generate bundle configuration for " + article(title) + " " + title,
		Long: fmt.Sprintf(`Generate bundle configuration for an existing %s.

The %s is read from the workspace and written as a bundle resource to
<config-dir>/<key>.%s.yml so that you can deploy it to other targets or manage
it as code. --existing-id takes the identifier that "databricks bundle deployment
bind" uses for this resource type.

Example:
  databricks bundle generate %s --existing-id ID --key my_%s`,
			title, title, desc.SingularName, use, desc.SingularName),
	}

	cmd.Flags().StringVar(&opts.id, "existing-id", "", "ID of the "+title+" to generate configuration for")
	cmd.MarkFlagRequired("existing-id")
	cmd.Flags().StringVarP(&opts.configDir, "config-dir", "d", "resources", `directory to write the configuration to`)
	cmd.Flags().BoolVarP(&opts.force, "force", "f", false, `force overwrite existing files in the output directory`)
	cmd.Flags().BoolVarP(&bind, "bind", "b", false, `automatically bind the generated resource to the existing resource`)
	cmd.Flags().MarkHidden("bind")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts.key = cmd.Flag("key").Value.String()
		key, err := generateResource(ctx, b, group, desc, opts)
		if err != nil {
			return err
		}

		if bind {
			return deployment.BindResource(cmd, key, opts.id, true, false, true)
		}
		return nil
	}

	return cmd
}

// generateResource reads the resource, writes it as <configDir>/<key>.<type>.yml under
// the bundle root, and returns the resource key it used.
func generateResource(ctx context.Context, b *bundle.Bundle, group string, desc resources.ResourceDescription, opts resourceOptions) (string, error) {
	resource, err := readResource(ctx, b.WorkspaceClient(ctx), group, opts.id)
	if err != nil {
		return "", err
	}

	key := opts.key
	if key == "" {
		key = textutil.NormalizeString(resource.GetName())
	}
	if key == "" {
		return "", fmt.Errorf("cannot derive a resource key for %s %q, pass --key", desc.SingularName, opts.id)
	}

	v, err := generate.ConvertResourceToValue(resource)
	if err != nil {
		return "", err
	}

	configDir := opts.configDir
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(b.BundleRootPath, configDir)
	}
	filename := filepath.Join(configDir, key+"."+desc.SingularName+".yml")

	result := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			group: dyn.V(map[string]dyn.Value{
				key: v,
			}),
		}),
	}
	err = yamlsaver.NewSaver().SaveAsYAML(result, filename, opts.force)
	if err != nil {
		return "", err
	}

	cmdio.LogString(ctx, desc.SingularTitle+" configuration successfully saved to "+filepath.ToSlash(filename))
	warnIfNotIncluded(ctx, b, filename)
	return key, nil
}

// readResource reads a resource through its deployment-engine adapter and maps the
// engine state onto the bundle resource type via JSON. The mapping is only trusted
// when feeding the result back through PrepareState reproduces the state; a type
// whose state is shaped differently from its configuration (secret scopes, for
// example) is rejected instead of being written as YAML that would not deploy.
func readResource(ctx context.Context, w *databricks.WorkspaceClient, group, id string) (config.ConfigResource, error) {
	adapter, err := dresources.NewAdapter(dresources.SupportedResources[group], group, w)
	if err != nil {
		return nil, err
	}
	remote, err := adapter.DoRead(ctx, id)
	if err != nil {
		return nil, err
	}
	state, err := adapter.RemapState(remote)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	resource, err := newConfigResource(group)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, resource)
	if err != nil {
		return nil, err
	}

	again, err := adapter.PrepareState(resource)
	if err != nil {
		return nil, err
	}
	check, err := json.Marshal(again)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, check) {
		return nil, fmt.Errorf("generating %s is not supported yet: the deployment engine state does not map onto bundle configuration", group)
	}
	return resource, nil
}

// newConfigResource returns a zero bundle resource of the type config.Resources
// declares for group.
func newConfigResource(group string) (config.ConfigResource, error) {
	for field := range reflect.TypeFor[config.Resources]().Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == group {
			return reflect.New(field.Type.Elem().Elem()).Interface().(config.ConfigResource), nil
		}
	}
	return nil, fmt.Errorf("internal error: no bundle resource type for %q", group)
}

// article picks the indefinite article for a lowercase title such as "experiment".
func article(title string) string {
	switch title[0] {
	case 'a', 'e', 'i', 'o', 'u':
		return "an"
	}
	return "a"
}
