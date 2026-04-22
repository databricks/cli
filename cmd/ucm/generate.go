package ucm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/ucm/config/generate"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

// defaultGenerateTarget is the target folder `ucm generate` seeds state under.
// Matches phases.SelectDefaultTarget's default so a subsequent `ucm deploy`
// against the freshly-emitted ucm.yml picks up the seeded state.
const defaultGenerateTarget = "default"

// generateClientFactory is the indirection tests override to inject an
// in-memory direct.Client. Production callers use the default, which reads
// the workspace client installed by root.MustWorkspaceClient off ctx.
var generateClientFactory = defaultGenerateClientFactory

// generateHostAndClient returns the host URL + direct.Client for the
// current command. Separate from generateClientFactory so tests can stand in
// a client without having to fake the cmdctx-resolved workspace client.
func defaultGenerateClientFactory(cmd *cobra.Command) (string, direct.Client, error) {
	ctx := cmd.Context()
	w := cmdctx.WorkspaceClient(ctx)
	if w == nil {
		return "", nil, fmt.Errorf("workspace client not configured")
	}
	host := ""
	if w.Config != nil {
		host = w.Config.Host
	}
	return host, direct.NewClient(w), nil
}

func newGenerateCommand() *cobra.Command {
	var outputDir string
	var kindsCSV string
	var name string
	var force bool

	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Scan an existing account+metastore+workspace and emit ucm.yml + seed state.",
		Long: `Scan an existing account+metastore+workspace and emit ucm.yml + seed state.

Points at a workspace and writes a starter ucm.yml plus seed direct-engine
state so a subsequent ` + "`ucm deploy`" + ` is a no-op instead of trying to
recreate everything.

Scanned kinds (default): catalog, schema, storage_credential,
external_location, volume, connection. Grants are intentionally excluded —
they reconcile per-securable at deploy time, so seeding them adds noise
without improving idempotency.

Known limitations:

  - Credentials with secret material (Azure service-principal ClientSecret)
    cannot round-trip — UC does not echo the secret. The generated YAML has
    a placeholder and a warning is printed; fill the secret in before the
    next deploy.

  - Grants: skipped. Re-emit them with ` + "`ucm deployment bind`" + ` after
    generation.

  - Catalog→schema tag inheritance is not reconstructed; tags are emitted
    as-declared on each resource. Clean up by hand if desired.

Examples:
  databricks ucm generate --name prod
  databricks ucm generate --output-dir ./bootstrap --kinds catalog,schema
  databricks ucm generate --force   # overwrite existing ucm.yml`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write ucm.yml + seed state into.")
	cmd.Flags().StringVar(&kindsCSV, "kinds", "", "Comma-separated resource kinds to scan. Default: catalog,schema,storage_credential,external_location,volume,connection.")
	cmd.Flags().StringVar(&name, "name", "", "Name for the ucm.yml ucm.name field. Defaults to a sanitized host-derived label.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite an existing ucm.yml in --output-dir.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		host, client, err := generateClientFactory(cmd)
		if err != nil {
			return err
		}

		kinds, err := parseKinds(kindsCSV)
		if err != nil {
			return err
		}

		effectiveName := name
		if effectiveName == "" {
			effectiveName = deriveName(host)
		}

		result, err := phases.Generate(ctx, client, phases.GenerateOptions{
			Name:  effectiveName,
			Host:  host,
			Kinds: kinds,
		})
		if err != nil {
			return fmt.Errorf("scan workspace: %w", err)
		}

		absOut, err := filepath.Abs(outputDir)
		if err != nil {
			return fmt.Errorf("resolve output dir: %w", err)
		}
		if err := os.MkdirAll(absOut, 0o755); err != nil {
			return fmt.Errorf("create output dir: %w", err)
		}

		ucmPath := filepath.Join(absOut, "ucm.yml")
		if err := generate.SaveToFile(result.Root, ucmPath, force); err != nil {
			return err
		}

		statePath := filepath.Join(absOut, filepath.FromSlash(deploy.LocalCacheDir), defaultGenerateTarget, direct.StateFileName)
		if err := direct.SaveState(statePath, result.State); err != nil {
			return fmt.Errorf("seed direct state: %w", err)
		}

		// Emit warnings before the success summary so they aren't lost in
		// scrollback when there are many resources.
		for _, wmsg := range result.Warnings {
			cmdio.LogString(ctx, "warning: "+wmsg)
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote %s", filepath.ToSlash(ucmPath)))
		cmdio.LogString(ctx, fmt.Sprintf("Seeded direct-engine state at %s", filepath.ToSlash(statePath)))
		cmdio.LogString(ctx, summarize(result))
		return nil
	}

	return cmd
}

// parseKinds splits a comma-separated kind list into a slice. Empty input
// returns nil so phases.Generate picks its default set.
func parseKinds(csv string) ([]string, error) {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil, nil
	}
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

// deriveName produces a plausible ucm.name from the workspace host. A host
// of https://acme-prod.cloud.databricks.com becomes "acme-prod". Falls back
// to "scanned" when the host is unparseable.
func deriveName(host string) string {
	h := host
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	if i := strings.Index(h, "."); i > 0 {
		h = h[:i]
	}
	h = strings.TrimSpace(h)
	if h == "" {
		return "scanned"
	}
	return h
}

// summarize returns a one-line counts breakdown so the user sees scale at a
// glance without having to grep through the YAML.
func summarize(r *phases.GenerateResult) string {
	counts := map[string]int{
		"catalogs":            len(r.Root.Resources.Catalogs),
		"schemas":             len(r.Root.Resources.Schemas),
		"storage_credentials": len(r.Root.Resources.StorageCredentials),
		"external_locations":  len(r.Root.Resources.ExternalLocations),
		"volumes":             len(r.Root.Resources.Volumes),
		"connections":         len(r.Root.Resources.Connections),
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(counts))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, counts[k]))
	}
	return "Scanned: " + strings.Join(parts, " ")
}
