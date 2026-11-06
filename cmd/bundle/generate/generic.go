package generate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
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

// resourceSpecs lists the resource types served by the generic generate command,
// one per resource family. The parity test in cmd/bundle checks it against
// config.SupportedResources.
var resourceSpecs []resourceSpec

// NewResourceCommands returns a generate subcommand for every entry in resourceSpecs.
func NewResourceCommands() []*cobra.Command {
	cmds := make([]*cobra.Command, 0, len(resourceSpecs))
	for _, spec := range resourceSpecs {
		cmds = append(cmds, newResourceCommand(spec))
	}
	return cmds
}

// resource describes a resource type that is read with a single API call and has
// no files to download. R is the API response type, C the bundle resource type in
// bundle/config/resources that R is narrowed to. Build a resourceSpec from it with
// newResourceSpec.
type resource[R, C any] struct {
	// LookupFlag names the flag that identifies the existing resource, for
	// example "existing-catalog-name". Its value is also what --bind binds to.
	LookupFlag  string
	LookupUsage string
	// Fetch reads the resource identified by the lookup value.
	Fetch func(context.Context, *databricks.WorkspaceClient, string) (R, error)
	// SkipFields are dropped from the generated YAML in addition to the
	// CLI-managed fields. Use it for server-populated fields the create API rejects.
	SkipFields []string
}

// bundleResource is the part of config.ConfigResource the generic command needs.
type bundleResource interface {
	ResourceDescription() resources.ResourceDescription
	GetName() string
}

// resourceSpec is the type-erased form of resource that resourceSpecs stores.
type resourceSpec struct {
	description resources.ResourceDescription
	lookupFlag  string
	lookupUsage string
	fetch       func(context.Context, *databricks.WorkspaceClient, string) (any, error)
	// convert narrows the response to the bundle resource and returns its
	// serialized form together with its in-product name.
	convert func(any) (dyn.Value, string, error)
}

func newResourceSpec[R, C any, PC interface {
	*C
	bundleResource
}](r resource[R, C]) resourceSpec {
	var zero C
	return resourceSpec{
		description: PC(&zero).ResourceDescription(),
		lookupFlag:  r.LookupFlag,
		lookupUsage: r.LookupUsage,
		fetch: func(ctx context.Context, w *databricks.WorkspaceClient, lookup string) (any, error) {
			return r.Fetch(ctx, w, lookup)
		},
		convert: func(response any) (dyn.Value, string, error) {
			c, err := generate.FromResponse[C](response)
			if err != nil {
				return dyn.InvalidValue, "", err
			}
			v, err := generate.ConvertResourceToValue(c, r.SkipFields...)
			if err != nil {
				return dyn.InvalidValue, "", err
			}
			return v, PC(&c).GetName(), nil
		},
	}
}

// resourceOptions carries the flag values of a generic generate subcommand.
type resourceOptions struct {
	lookup    string
	key       string
	configDir string
	force     bool
}

func newResourceCommand(spec resourceSpec) *cobra.Command {
	var opts resourceOptions
	var bind bool

	name := spec.description.SingularName
	title := spec.description.SingularTitle
	cmd := &cobra.Command{
		Use:   name,
		Short: "Generate bundle configuration for " + article(title) + " " + title,
		Long: fmt.Sprintf(`Generate bundle configuration for an existing %s.

This command reads the %s from the workspace and writes it as a bundle resource
to <config-dir>/<key>.%s.yml so that you can deploy it to other targets or
manage it as code.

Examples:
  databricks bundle generate %s --%s NAME --key my_%s`,
			title, title, name, name, spec.lookupFlag, name),
	}

	cmd.Flags().StringVar(&opts.lookup, spec.lookupFlag, "", spec.lookupUsage)
	cmd.MarkFlagRequired(spec.lookupFlag)
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
		key, err := generateResource(ctx, b, spec, opts)
		if err != nil {
			return err
		}

		if bind {
			return deployment.BindResource(cmd, key, opts.lookup, true, false, true)
		}
		return nil
	}

	return cmd
}

// generateResource fetches the resource, writes it as <configDir>/<key>.<type>.yml
// under the bundle root, and returns the resource key it used.
func generateResource(ctx context.Context, b *bundle.Bundle, spec resourceSpec, opts resourceOptions) (string, error) {
	response, err := spec.fetch(ctx, b.WorkspaceClient(ctx), opts.lookup)
	if err != nil {
		return "", err
	}
	v, name, err := spec.convert(response)
	if err != nil {
		return "", err
	}

	key := opts.key
	if key == "" {
		key = textutil.NormalizeString(name)
	}
	if key == "" {
		return "", fmt.Errorf("cannot derive a resource key for %s %q, pass --key", spec.description.SingularName, opts.lookup)
	}

	configDir := opts.configDir
	if !filepath.IsAbs(configDir) {
		configDir = filepath.Join(b.BundleRootPath, configDir)
	}
	filename := filepath.Join(configDir, key+"."+spec.description.SingularName+".yml")

	result := map[string]dyn.Value{
		"resources": dyn.V(map[string]dyn.Value{
			spec.description.PluralName: dyn.V(map[string]dyn.Value{
				key: v,
			}),
		}),
	}
	err = yamlsaver.NewSaver().SaveAsYAML(result, filename, opts.force)
	if err != nil {
		return "", err
	}

	cmdio.LogString(ctx, spec.description.SingularTitle+" configuration successfully saved to "+filepath.ToSlash(filename))
	warnIfNotIncluded(ctx, b, filename)
	return key, nil
}

// article picks the indefinite article for a title such as "Experiment".
func article(title string) string {
	switch title[0] {
	case 'A', 'E', 'I', 'O', 'U':
		return "an"
	}
	return "a"
}
