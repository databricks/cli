package ucm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	ucmtemplates "github.com/databricks/cli/ucm/templates"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [TEMPLATE_PATH]",
		Short: "Scaffold a new ucm.yml project from a starter template.",
		Long: fmt.Sprintf(`Scaffold a new ucm.yml project from a starter template.

TEMPLATE_PATH is optional. It can be one of:
%s
- a local filesystem path to a ucm template directory
- a Git repository URL (https:// or git@)

Examples:
  databricks ucm init                   # Prompt for a built-in template
  databricks ucm init default           # Minimal catalog + schema + grant
  databricks ucm init brownfield        # Stub for 'ucm generate' follow-up
  databricks ucm init multienv          # dev/staging/prod targets
  databricks ucm init ./my-template     # Initialize from a local directory
  databricks ucm init --output-dir ./my-project default

See https://docs.databricks.com/en/dev-tools/ucm/index.html for more
information on ucm templates.`, ucmtemplates.HelpDescriptions()),
		Args: root.MaximumNArgs(1),
	}

	var configFile string
	var outputDir string
	var templateDir string
	var tag string
	var branch string
	cmd.Flags().StringVar(&configFile, "config-file", "", "JSON file containing key value pairs of input parameters required for template initialization.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory path within a Git repository containing the template.")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to.")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to use for template initialization.")
	cmd.Flags().StringVar(&tag, "tag", "", "Git tag to use for template initialization.")

	// The template renderer exposes helpers (workspace_host, short_name, etc.)
	// that need a live workspace client even when a given template doesn't call
	// them. Mirror DAB's init and eagerly acquire one so any template — custom
	// or built-in — can reference the full helper surface.
	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if tag != "" && branch != "" {
			return errors.New("only one of --tag or --branch can be specified")
		}

		var templatePathOrUrl string
		if len(args) > 0 {
			templatePathOrUrl = args[0]
		}

		ctx := cmd.Context()

		// If the user selected a built-in ucm template, stage it to a temp dir
		// and hand the path to the shared resolver so all custom-template
		// plumbing (local filer, schema prompt, rendering) is reused as-is.
		pathOrUrl, cleanup, err := resolveBuiltinOrPassthrough(ctx, templatePathOrUrl)
		if err != nil {
			return err
		}
		defer cleanup()

		r := template.Resolver{
			TemplatePathOrUrl: pathOrUrl,
			ConfigFile:        configFile,
			OutputDir:         outputDir,
			TemplateDir:       templateDir,
			Tag:               tag,
			Branch:            branch,
		}

		tmpl, err := r.Resolve(ctx)
		if errors.Is(err, template.ErrCustomSelected) {
			cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
			cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/ucm/index.html to learn more about ucm templates.")
			return nil
		}
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		if err := tmpl.Writer.Materialize(ctx, tmpl.Reader); err != nil {
			return err
		}
		tmpl.Writer.LogTelemetry(ctx)
		return nil
	}

	return cmd
}

// resolveBuiltinOrPassthrough returns (pathOrUrl, cleanup, err). If the input
// matches a built-in ucm template, the embedded FS is extracted to a temp dir
// and that path is returned; otherwise the original value is passed through
// for the shared resolver to handle as a local path or git URL.
//
// Extraction to disk is the simplest way to plug the ucm-owned embed.FS into
// libs/template without forking the resolver — template.Resolver already
// understands local directories via NewLocalReader.
func resolveBuiltinOrPassthrough(ctx context.Context, input string) (string, func(), error) {
	noop := func() {}

	// Interactive selection from the built-in list when no argument is given.
	if input == "" {
		if !cmdio.IsPromptSupported(ctx) {
			// Nothing to expand; let the shared resolver emit its own error.
			return "", noop, nil
		}
		options := make([]cmdio.Tuple, 0, len(ucmtemplates.List()))
		for _, b := range ucmtemplates.List() {
			options = append(options, cmdio.Tuple{Name: b.Name, Id: b.Description})
		}
		name, err := cmdio.SelectOrdered(ctx, options, "Template to use")
		if err != nil {
			return "", noop, err
		}
		// SelectOrdered returns the Id (description); map back to the name.
		input = descriptionToName(name)
	}

	reader := ucmtemplates.Lookup(input)
	if reader == nil {
		return input, noop, nil
	}

	dir, err := extractBuiltin(ctx, reader, input)
	if err != nil {
		return "", noop, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// descriptionToName reverses ucmtemplates.List's (name -> description) mapping
// so cmdio.SelectOrdered's choice can be routed back to the embed entry.
func descriptionToName(description string) string {
	for _, b := range ucmtemplates.List() {
		if b.Description == description || b.Name == description {
			return b.Name
		}
	}
	return description
}

// extractBuiltin writes the embedded template filesystem to a newly created
// temp directory so template.Resolver.Resolve can consume it as a local path.
func extractBuiltin(ctx context.Context, reader interface {
	SchemaFS(context.Context) (fs.FS, error)
}, name string) (string, error) {
	src, err := reader.SchemaFS(ctx)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp("", "ucm-init-"+name+"-*")
	if err != nil {
		return "", err
	}

	if err := copyFS(src, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

// copyFS copies a read-only filesystem tree to a destination on disk.
func copyFS(src fs.FS, dst string) error {
	return fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, filepath.FromSlash(path))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := src.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		if _, err := io.Copy(out, in); err != nil {
			return err
		}
		return nil
	})
}
