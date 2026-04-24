package ucm

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// notDeployedURL is the literal rendered when a URL-bearing resource has no
// ID in the local tfstate. Matches the DAB wording at
// bundle/render/render_text_output.go so users reading both tools' output
// get a consistent signal.
const notDeployedURL = "(not deployed)"

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize resources declared by this ucm deployment.",
		Long: `Summarize the resources declared by this ucm deployment, grouped by kind,
with workspace URLs when a Workspace.Host is configured.

Mirrors ` + "`databricks bundle summary`" + `: loads the per-target
terraform.tfstate from the local cache to determine which resources have
actually been deployed. URL lines show the workspace console link for
resources present in state and ` + "`" + notDeployedURL + "`" + ` for resources declared in
ucm.yml but not yet applied. Run ` + "`ucm deploy`" + ` to realize declared intents.

Common invocations:
  databricks ucm summary                   # Text summary of the default target
  databricks ucm summary --target prod     # Summary of a named target
  databricks ucm summary -o json           # Emit the full config as JSON`,
		Args:    root.NoArgs,
		PreRunE: utils.MustWorkspaceClient,
	}

	// forcePull is accepted for DAB parity but is a no-op today: summary reads
	// the local tfstate, not the remote workspace. Wiring a real state pull
	// belongs in a separate change.
	var forcePull bool
	var includeLocations bool
	var showFullConfig bool
	cmd.Flags().BoolVar(&forcePull, "force-pull", false, "Skip local cache and load the state from the remote workspace (no-op today)")
	cmd.Flags().BoolVar(&includeLocations, "include-locations", false, "Include location information in the output")
	_ = cmd.Flags().MarkHidden("include-locations")
	cmd.Flags().BoolVar(&showFullConfig, "show-full-config", false, "Load and output the full ucm config")
	_ = cmd.Flags().MarkHidden("show-full-config")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u := utils.ProcessUcm(cmd, utils.ProcessOptions{InitIDs: true})
		ctx := cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		if includeLocations {
			ucm.ApplyContext(ctx, u, mutator.PopulateLocations())
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
		}

		out := cmd.OutOrStdout()
		if showFullConfig {
			buf, err := json.MarshalIndent(u.Config, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
			return nil
		}
		switch summaryOutputType(cmd) {
		case flags.OutputJSON:
			buf, err := json.MarshalIndent(u.Config, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
			return nil
		default:
			renderSummaryText(out, u)
			return nil
		}
	}

	return cmd
}

// summaryOutputType mirrors planOutputType: returns OutputText when the
// persistent --output flag wasn't wired (unit tests build the tree directly
// via New() rather than going through root.New).
func summaryOutputType(cmd *cobra.Command) flags.Output {
	if cmd.Flag("output") == nil {
		return flags.OutputText
	}
	return root.OutputType(cmd)
}

// resourceRow is one line in a summary group.
type resourceRow struct {
	Key  string
	Name string
	URL  string
}

// resourceGroup is a titled collection of resourceRows (e.g. "Catalogs").
// HasURL distinguishes kinds that carry a workspace URL (so an empty URL
// renders as "(not deployed)") from kinds that never do (Grants,
// TagValidationRules) — those stay URL-less regardless of deploy state.
type resourceGroup struct {
	Title  string
	Rows   []resourceRow
	HasURL bool
}

// renderSummaryText writes the bundle-summary-shaped text output: header
// (Name / Target / Workspace) followed by one section per non-empty resource
// group. Empty groups are suppressed.
func renderSummaryText(out io.Writer, u *ucm.Ucm) {
	renderSummaryHeader(out, u)

	groups := collectResourceGroups(&u.Config)
	cyan := color.New(color.FgCyan).SprintFunc()
	for _, g := range groups {
		fmt.Fprintf(out, "%s:\n", g.Title)
		for _, r := range g.Rows {
			fmt.Fprintf(out, "  %s:\n", r.Key)
			fmt.Fprintf(out, "    Name: %s\n", r.Name)
			if !g.HasURL {
				continue
			}
			if r.URL == "" {
				fmt.Fprintf(out, "    URL:  %s\n", notDeployedURL)
			} else {
				fmt.Fprintf(out, "    URL:  %s\n", cyan(r.URL))
			}
		}
	}
}

func renderSummaryHeader(out io.Writer, u *ucm.Ucm) {
	bold := color.New(color.Bold).SprintFunc()
	cfg := &u.Config
	if cfg.Ucm.Name != "" {
		fmt.Fprintf(out, "Name: %s\n", bold(cfg.Ucm.Name))
	}
	if cfg.Ucm.Target != "" {
		fmt.Fprintf(out, "Target: %s\n", bold(cfg.Ucm.Target))
	}

	var userName string
	if u.CurrentUser != nil && u.CurrentUser.User != nil {
		userName = u.CurrentUser.UserName
	}
	hasWorkspace := cfg.Workspace.Host != "" || userName != "" || cfg.Workspace.RootPath != ""
	if hasWorkspace {
		fmt.Fprintln(out, "Workspace:")
		if cfg.Workspace.Host != "" {
			fmt.Fprintf(out, "  Host: %s\n", bold(cfg.Workspace.Host))
		}
		if userName != "" {
			fmt.Fprintf(out, "  User: %s\n", bold(userName))
		}
		if cfg.Workspace.RootPath != "" {
			fmt.Fprintf(out, "  Path: %s\n", bold(cfg.Workspace.RootPath))
		}
	}
	fmt.Fprintln(out)
}

// collectResourceGroups gathers the declared resources into titled groups
// sorted by title, each group's rows sorted by key. Groups with no entries
// are omitted so the output only shows sections that exist.
//
// URL values are read from the config fields populated by
// mutator.InitializeURLs — an empty URL means the resource is declared but
// not yet deployed, and is rendered as "(not deployed)" by renderSummaryText.
func collectResourceGroups(cfg *config.Root) []resourceGroup {
	var groups []resourceGroup

	if len(cfg.Resources.Catalogs) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.Catalogs))
		for key, c := range cfg.Resources.Catalogs {
			rows = append(rows, resourceRow{
				Key:  key,
				Name: c.Name,
				URL:  c.URL,
			})
		}
		groups = append(groups, resourceGroup{Title: "Catalogs", Rows: rows, HasURL: true})
	}

	if len(cfg.Resources.Schemas) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.Schemas))
		for key, s := range cfg.Resources.Schemas {
			full := s.Name
			if s.Catalog != "" {
				full = s.Catalog + "." + s.Name
			}
			rows = append(rows, resourceRow{Key: key, Name: full, URL: s.URL})
		}
		groups = append(groups, resourceGroup{Title: "Schemas", Rows: rows, HasURL: true})
	}

	if len(cfg.Resources.Volumes) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.Volumes))
		for key, v := range cfg.Resources.Volumes {
			full := v.Name
			if v.CatalogName != "" && v.SchemaName != "" {
				full = v.CatalogName + "." + v.SchemaName + "." + v.Name
			}
			rows = append(rows, resourceRow{Key: key, Name: full, URL: v.URL})
		}
		groups = append(groups, resourceGroup{Title: "Volumes", Rows: rows, HasURL: true})
	}

	if len(cfg.Resources.StorageCredentials) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.StorageCredentials))
		for key, sc := range cfg.Resources.StorageCredentials {
			rows = append(rows, resourceRow{
				Key:  key,
				Name: sc.Name,
				URL:  sc.URL,
			})
		}
		groups = append(groups, resourceGroup{Title: "Storage credentials", Rows: rows, HasURL: true})
	}

	if len(cfg.Resources.ExternalLocations) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.ExternalLocations))
		for key, el := range cfg.Resources.ExternalLocations {
			rows = append(rows, resourceRow{
				Key:  key,
				Name: el.Name,
				URL:  el.URL,
			})
		}
		groups = append(groups, resourceGroup{Title: "External locations", Rows: rows, HasURL: true})
	}

	if len(cfg.Resources.Connections) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.Connections))
		for key, conn := range cfg.Resources.Connections {
			rows = append(rows, resourceRow{
				Key:  key,
				Name: conn.Name,
				URL:  conn.URL,
			})
		}
		groups = append(groups, resourceGroup{Title: "Connections", Rows: rows, HasURL: true})
	}

	if len(cfg.Resources.Grants) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.Grants))
		for key, g := range cfg.Resources.Grants {
			// Grants have no workspace URL; summarise securable + principal.
			name := fmt.Sprintf("%s %s -> %s", g.Securable.Type, g.Securable.Name, g.Principal)
			rows = append(rows, resourceRow{Key: key, Name: name})
		}
		groups = append(groups, resourceGroup{Title: "Grants", Rows: rows})
	}

	if len(cfg.Resources.TagValidationRules) > 0 {
		rows := make([]resourceRow, 0, len(cfg.Resources.TagValidationRules))
		for key := range cfg.Resources.TagValidationRules {
			rows = append(rows, resourceRow{Key: key, Name: key})
		}
		groups = append(groups, resourceGroup{Title: "Tag validation rules", Rows: rows})
	}

	slices.SortFunc(groups, func(a, b resourceGroup) int { return cmp.Compare(a.Title, b.Title) })
	for i := range groups {
		slices.SortFunc(groups[i].Rows, func(a, b resourceRow) int { return cmp.Compare(a.Key, b.Key) })
	}
	return groups
}
