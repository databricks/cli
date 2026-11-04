package generate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/cli/libs/textutil"
	"github.com/spf13/cobra"
)

const initBareFlag = "init-bare"

// configureBundle sets up diagnostics and loads the bundle a generate command writes into.
//
// Without a bundle it fails, unless --init-bare is set: then it materializes the
// default-bare template into the working directory and loads that. bundleOnlyFlags
// name flags that only make sense against an existing bundle (bind, resource, watch);
// they are rejected before anything is written to disk.
func configureBundle(cmd *cobra.Command, bundleOnlyFlags ...string) (*bundle.Bundle, error) {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	b := root.TryConfigureBundle(cmd)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}
	if b != nil {
		return b, nil
	}

	initBare, err := cmd.Flags().GetBool(initBareFlag)
	if err != nil {
		return nil, err
	}
	if !initBare {
		return nil, errors.New("no databricks.yml found in this directory or any parent; run `databricks bundle init default-bare` first or re-run with --init-bare")
	}
	for _, name := range bundleOnlyFlags {
		if cmd.Flags().Changed(name) {
			return nil, fmt.Errorf("--%s requires an existing bundle and cannot be combined with --%s", name, initBareFlag)
		}
	}

	// The template renders workspace_host from the workspace client, which the
	// generators otherwise take from the bundle that does not exist yet.
	err = root.MustWorkspaceClient(cmd, nil)
	if err != nil {
		return nil, err
	}
	err = initBareBundle(cmd.Context())
	if err != nil {
		return nil, err
	}

	b = root.MustConfigureBundle(cmd)
	if b == nil || logdiag.HasError(cmd.Context()) {
		return nil, root.ErrAlreadyPrinted
	}
	return b, nil
}

// initBareBundle materializes the default-bare template into the working directory,
// naming the bundle after it.
func initBareBundle(ctx context.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	name := textutil.NormalizeString(filepath.Base(cwd))
	if name == "" {
		name = "my_bundle"
	}

	configFile, err := os.CreateTemp("", "bundle-generate-init-bare-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(configFile.Name())
	_, err = fmt.Fprintf(configFile, "{\"project_name\":%q}", name)
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
