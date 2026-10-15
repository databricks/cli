package generate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/cli/libs/textutil"
	"github.com/spf13/cobra"
)

// ensureGenerateBundle configures a bundle for generation. If no bundle exists and
// --init-bare is set, it materializes a bare template first. If no bundle exists
// and --init-bare is not set, it returns an error unless requireExistingErr is nil.
func ensureGenerateBundle(cmd *cobra.Command, requireExistingErr error) (*bundleContext, error) {
	ctx := initGenerateContext(cmd)
	b := root.TryConfigureBundle(cmd)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	if b != nil {
		return &bundleContext{hadExistingBundle: true}, nil
	}

	if requireExistingErr != nil {
		return nil, requireExistingErr
	}

	initBare, err := cmd.Flags().GetBool("init-bare")
	if err != nil {
		return nil, err
	}
	if !initBare {
		return nil, errors.New("no databricks.yml found in this directory or any parent. Run `databricks bundle init default-bare` first or re-run this command with --init-bare")
	}

	err = initializeBareBundle(ctx)
	if err != nil {
		return nil, err
	}

	b = root.MustConfigureBundle(cmd)
	if b == nil || logdiag.HasError(cmd.Context()) {
		return nil, root.ErrAlreadyPrinted
	}

	return &bundleContext{hadExistingBundle: false}, nil
}

// initializeBareBundle materializes a bare bundle template in the current directory.
func initializeBareBundle(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	projectName := textutil.NormalizeString(filepath.Base(cwd))
	if projectName == "" {
		projectName = "my_bundle"
	}

	configFile, err := os.CreateTemp("", "databricks-bundle-generate-init-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(configFile.Name())

	_, err = configFile.WriteString(fmt.Sprintf("{\"project_name\":%q}", projectName))
	if err != nil {
		configFile.Close()
		return err
	}
	err = configFile.Close()
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "No databricks.yml found. Initializing a bare bundle skeleton.")

	r := template.Resolver{
		TemplatePathOrUrl: string(template.DefaultBare),
		ConfigFile:        configFile.Name(),
		OutputDir:         cwd,
	}

	tmpl, err := r.Resolve(ctx)
	if err != nil {
		return err
	}
	defer tmpl.Reader.Cleanup(ctx)

	err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
	if err != nil {
		return err
	}
	tmpl.Writer.LogTelemetry(ctx)
	return nil
}

// initGenerateContext sets up the context for generation commands with logging diagnostics.
func initGenerateContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}
	return ctx
}

// bundleContext tracks whether a bundle existed before generate was run.
type bundleContext struct {
	hadExistingBundle bool
}
