package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

// listSkillsFn is the function used to render the skills list.
// It is a package-level var so tests can replace the data-fetching layer.
var listSkillsFn = defaultListSkills

func NewListCmd() *cobra.Command {
	var scopeFlag string
	var projectFlag, globalFlag bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed AI tools components",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Reject the legacy --project --global combination here so it
			// doesn't silently degrade to --scope=both. Users who want both
			// scopes should use --scope=both (the new explicit spelling).
			if projectFlag && globalFlag && scopeFlag == "" {
				return errors.New("cannot use --global and --project together")
			}

			projectFlag, globalFlag, err := parseScopeFlag(scopeFlag, projectFlag, globalFlag, true)
			if err != nil {
				return err
			}

			// list: empty scope = show both. --scope=both also lands here.
			var scope string
			switch {
			case projectFlag && !globalFlag:
				scope = installer.ScopeProject
			case globalFlag && !projectFlag:
				scope = installer.ScopeGlobal
			}
			return listSkillsFn(cmd, scope)
		},
	}

	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Scope to show: project, global, or both (default: both)")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Show only project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Show only globally-scoped skills")
	markScopeBoolsDeprecated(cmd)
	return cmd
}

// listOutput is the structured representation of `aitools list` used by both
// text rendering and `--output json` consumers. The JSON shape is part of
// the public CLI contract; do not break field names or types.
type listOutput struct {
	Release string                  `json:"release"`
	Skills  []skillEntry            `json:"skills"`
	Summary map[string]scopeSummary `json:"summary"`
}

type skillEntry struct {
	Name          string            `json:"name"`
	LatestVersion string            `json:"latest_version"`
	Experimental  bool              `json:"experimental"`
	Installed     map[string]string `json:"installed"`
}

type scopeSummary struct {
	Installed int `json:"installed"`
	Total     int `json:"total"`

	// loaded preserves text rendering semantics without changing the JSON contract.
	loaded bool
}

func defaultListSkills(cmd *cobra.Command, scope string) error {
	ctx := cmd.Context()

	out, err := buildListOutput(ctx, scope)
	if err != nil {
		return err
	}

	switch root.OutputType(cmd) {
	case flags.OutputJSON:
		return renderListJSON(cmd.OutOrStdout(), out)
	default:
		renderListText(ctx, out, scope)
		return nil
	}
}

// buildListOutput fetches the manifest and per-scope install state and
// returns the structured listOutput. scope=="" loads both scopes; "global"
// or "project" loads only that scope.
func buildListOutput(ctx context.Context, scope string) (listOutput, error) {
	ref, explicit, err := installer.GetSkillsRef(ctx)
	if err != nil {
		return listOutput{}, err
	}

	src := &installer.GitHubManifestSource{}
	manifest, ref, err := installer.FetchSkillsManifestWithFallback(ctx, src, ref, !explicit)
	if err != nil {
		return listOutput{}, fmt.Errorf("failed to fetch manifest: %w", err)
	}

	globalState := loadStateForScope(ctx, scope, installer.ScopeProject, installer.GlobalSkillsDir, "global")
	projectState := loadStateForScope(ctx, scope, installer.ScopeGlobal, installer.ProjectSkillsDir, "project")

	names := slices.Sorted(maps.Keys(manifest.Skills))

	out := listOutput{
		Release: strings.TrimPrefix(ref, "v"),
		Skills:  make([]skillEntry, 0, len(names)),
		Summary: map[string]scopeSummary{},
	}

	globalCount, projectCount := 0, 0
	for _, name := range names {
		meta := manifest.Skills[name]
		entry := skillEntry{
			Name:          name,
			LatestVersion: meta.Version,
			Experimental:  meta.IsExperimental(),
			Installed:     map[string]string{},
		}
		if globalState != nil {
			if v, ok := globalState.Skills[name]; ok {
				entry.Installed[installer.ScopeGlobal] = v
				globalCount++
			}
		}
		if projectState != nil {
			if v, ok := projectState.Skills[name]; ok {
				entry.Installed[installer.ScopeProject] = v
				projectCount++
			}
		}
		out.Skills = append(out.Skills, entry)
	}

	// Include a summary entry for every scope that was queried, even when the
	// install state is missing — agents should see "0/N" rather than guess
	// from the absence of a key.
	if scope != installer.ScopeProject {
		out.Summary[installer.ScopeGlobal] = scopeSummary{Installed: globalCount, Total: len(names), loaded: globalState != nil}
	}
	if scope != installer.ScopeGlobal {
		out.Summary[installer.ScopeProject] = scopeSummary{Installed: projectCount, Total: len(names), loaded: projectState != nil}
	}

	return out, nil
}

// loadStateForScope returns the install state for the named scope when the
// scope filter allows it. excludeScope is the scope value that means "skip
// loading this one" (so passing ScopeProject to the global loader skips
// global when --scope=project).
func loadStateForScope(ctx context.Context, scopeFilter, excludeScope string, dirFn func(context.Context) (string, error), label string) *installer.InstallState {
	if scopeFilter == excludeScope {
		return nil
	}
	dir, err := dirFn(ctx)
	if err != nil {
		return nil
	}
	state, err := installer.LoadState(dir)
	if err != nil {
		log.Debugf(ctx, "Could not load %s install state: %v", label, err)
		return nil
	}
	return state
}

func renderListJSON(w io.Writer, out listOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func renderListText(ctx context.Context, out listOutput, scope string) {
	cmdio.LogString(ctx, "Available skills (v"+out.Release+"):")
	cmdio.LogString(ctx, "")

	bothScopes := scope == "" &&
		out.Summary[installer.ScopeGlobal].loaded &&
		out.Summary[installer.ScopeProject].loaded

	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tVERSION\tINSTALLED")
	for _, s := range out.Skills {
		tag := ""
		if s.Experimental {
			tag = " [experimental]"
		}
		fmt.Fprintf(tw, "  %s%s\tv%s\t%s\n", s.Name, tag, s.LatestVersion, installedStatusFromEntry(s, bothScopes))
	}
	tw.Flush()
	cmdio.LogString(ctx, buf.String())

	cmdio.LogString(ctx, summaryLine(out, scope))
}

func installedStatusFromEntry(s skillEntry, bothScopes bool) string {
	globalVer := s.Installed[installer.ScopeGlobal]
	projectVer := s.Installed[installer.ScopeProject]

	if globalVer == "" && projectVer == "" {
		return "not installed"
	}

	if bothScopes && globalVer != "" && projectVer != "" {
		return versionLabel(projectVer, s.LatestVersion) + " (project, global)"
	}

	if projectVer != "" {
		label := versionLabel(projectVer, s.LatestVersion)
		if bothScopes {
			return label + " (project)"
		}
		return label
	}

	label := versionLabel(globalVer, s.LatestVersion)
	if bothScopes {
		return label + " (global)"
	}
	return label
}

// versionLabel formats version with update status.
func versionLabel(installed, latest string) string {
	if installed == latest {
		return "v" + installed + " (up to date)"
	}
	return "v" + installed + " (update available)"
}

func summaryLine(out listOutput, scope string) string {
	g, gOK := out.Summary[installer.ScopeGlobal]
	p, pOK := out.Summary[installer.ScopeProject]

	switch {
	case gOK && pOK:
		// Mirror prior behavior: only print the dual-scope line when both
		// scopes have a state file; otherwise only mention the one that does.
		if g.loaded && p.loaded {
			return fmt.Sprintf("%d/%d skills installed (global), %d/%d (project)", g.Installed, g.Total, p.Installed, p.Total)
		}
		if p.loaded {
			return fmt.Sprintf("%d/%d skills installed (project)", p.Installed, p.Total)
		}
		return fmt.Sprintf("%d/%d skills installed (global)", g.Installed, g.Total)
	case pOK:
		return fmt.Sprintf("%d/%d skills installed (project)", p.Installed, p.Total)
	default:
		return fmt.Sprintf("%d/%d skills installed (global)", g.Installed, g.Total)
	}
}
