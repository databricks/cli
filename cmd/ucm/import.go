package ucm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

// importKinds is the closed set of resource kinds the verb accepts. Kept as
// a slice (not a map) so help text lists them in a stable order.
var importKinds = []phases.ImportKind{
	phases.ImportCatalog,
	phases.ImportSchema,
	phases.ImportStorageCredential,
	phases.ImportExternalLocation,
	phases.ImportVolume,
	phases.ImportConnection,
}

func newImportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import <kind> <name>",
		Short: "Bind state for an existing UC resource to a ucm.yml declaration.",
		Long: `Bind state for an existing UC resource to a ucm.yml declaration.

Supported <kind>: catalog, schema, storage_credential, external_location,
volume, connection.

<name> is the UC identifier — e.g. "sales_prod" for a catalog, "sales.raw"
for a schema, "sales.raw.docs" for a volume, or just the resource name for
storage_credential/external_location/connection.

The resource must already be declared under resources.<kind>.<key> in
ucm.yml. Use --key to specify which ucm.yml key to bind under when it
differs from the UC name (or the last path component for schemas/volumes).

Common invocations:
  databricks ucm import catalog sales_prod
  databricks ucm import schema sales_prod.raw --key raw_landing
  databricks ucm import storage_credential my_cred --auto-approve`,
		Args:    root.ExactArgs(2),
		PreRunE: utils.MustWorkspaceClient,
	}

	var key string
	var autoApprove bool
	cmd.Flags().StringVar(&key, "key", "", "ucm.yml map key to bind under (defaults to the UC name or its last path component).")
	cmd.Flags().BoolVar(&autoApprove, "auto-approve", false, "Skip the interactive confirmation prompt.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		kind, err := parseImportKind(args[0])
		if err != nil {
			return err
		}
		name := args[1]
		resolvedKey := key
		if resolvedKey == "" {
			resolvedKey = defaultImportKey(kind, name)
		}

		if !autoApprove {
			if !cmdio.IsPromptSupported(ctx) {
				return errors.New("please specify --auto-approve since terminal does not support interactive prompts")
			}
			prompt := fmt.Sprintf("Import %s %q into resources.%s.%s?", kind, name, pluralImportKind(kind), resolvedKey)
			ok, err := cmdio.AskYesOrNo(ctx, prompt)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("import aborted")
			}
		}

		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx = cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		opts, err := buildPhaseOptions(ctx, u)
		if err != nil {
			return fmt.Errorf("resolve import options: %w", err)
		}

		phases.Import(ctx, u, opts, phases.ImportRequest{Kind: kind, Name: name, Key: resolvedKey})
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Imported %s.%s (%s)\n", kind, resolvedKey, name)
		return nil
	}

	return cmd
}

// parseImportKind validates arg0 against the supported kinds and returns a
// typed value, or a helpful error listing the accepted values.
func parseImportKind(s string) (phases.ImportKind, error) {
	for _, k := range importKinds {
		if string(k) == s {
			return k, nil
		}
	}
	accepted := make([]string, len(importKinds))
	for i, k := range importKinds {
		accepted[i] = string(k)
	}
	return "", fmt.Errorf("unsupported kind %q (want one of: %s)", s, strings.Join(accepted, ", "))
}

// defaultImportKey falls back to the UC name for simple kinds and to the
// last path component for schemas/volumes (since their names carry the
// parent hierarchy as dot-separated segments).
func defaultImportKey(kind phases.ImportKind, name string) string {
	switch kind {
	case phases.ImportSchema, phases.ImportVolume:
		if i := strings.LastIndex(name, "."); i >= 0 {
			return name[i+1:]
		}
	}
	return name
}

// pluralImportKind is the resources.<plural> map name for kind. Kept here
// (not exported from phases) so the CLI layer owns user-facing strings.
func pluralImportKind(k phases.ImportKind) string {
	switch k {
	case phases.ImportCatalog:
		return "catalogs"
	case phases.ImportSchema:
		return "schemas"
	case phases.ImportStorageCredential:
		return "storage_credentials"
	case phases.ImportExternalLocation:
		return "external_locations"
	case phases.ImportVolume:
		return "volumes"
	case phases.ImportConnection:
		return "connections"
	}
	return string(k)
}
