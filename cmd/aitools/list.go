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
	"github.com/databricks/cli/libs/aitools/agents"
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
		Short: "List installed skills and plugins",
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
// the public CLI contract; do not break field names or types. The agents field
// is additive (omitempty) and does not affect the existing skills/summary shape.
type listOutput struct {
	Release string                  `json:"release"`
	Skills  []skillEntry            `json:"skills"`
	Summary map[string]scopeSummary `json:"summary"`
	Agents  []agentEntry            `json:"agents,omitempty"`
}

// agentEntry reports per-agent plugin state for `list`. It mirrors skillEntry:
// Installed maps scope -> the plugin recorded in that scope, so a stale scoped
// install stays visible next to an up-to-date one. Managed says whether the CLI
// installs and tracks the plugin. Up-to-date-ness is derived by comparing each
// Installed version against the top-level release, exactly as the skills view
// does, so there is no precomputed cross-scope status to keep in sync.
type agentEntry struct {
	Name      string                `json:"name"`
	Managed   bool                  `json:"managed"`
	Installed map[string]pluginInfo `json:"installed,omitempty"`
}

// pluginInfo is the per-scope plugin record surfaced in list output.
type pluginInfo struct {
	Version     string `json:"version,omitempty"`
	NativeScope string `json:"native_scope,omitempty"`
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
		Release: installer.DisplaySkillsVersion(ref),
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

	// Filter unloaded scopes here so buildAgentEntries can assume every state
	// it receives is non-nil.
	states := map[string]*installer.InstallState{}
	if globalState != nil {
		states[installer.ScopeGlobal] = globalState
	}
	if projectState != nil {
		states[installer.ScopeProject] = projectState
	}
	out.Agents = buildAgentEntries(states)

	return out, nil
}

// buildAgentEntries reports per-agent plugin state: each plugin agent with a
// recorded install (its version per scope). states maps scope -> install state
// and must contain only non-nil states; the caller filters scopes it did not
// load. Status across scopes is left for the renderer (and JSON consumers) to
// derive from the per-scope versions, so no cross-scope record is merged away here.
func buildAgentEntries(states map[string]*installer.InstallState) []agentEntry {
	var entries []agentEntry
	for _, a := range agents.Registry {
		if a.Plugin == nil {
			continue
		}

		installed := map[string]pluginInfo{}
		for scope, st := range states {
			if rec, ok := st.Plugins[a.Name]; ok {
				installed[scope] = pluginInfo{Version: rec.Version, NativeScope: rec.Scope}
			}
		}
		if len(installed) > 0 {
			entries = append(entries, agentEntry{Name: a.Name, Managed: true, Installed: installed})
		}
	}
	return entries
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
	bothScopes := scope == "" &&
		out.Summary[installer.ScopeGlobal].loaded &&
		out.Summary[installer.ScopeProject].loaded

	// Split experimental skills into their own group so they are clearly
	// separated from the stable set rather than interleaved alphabetically.
	var stable, experimental []skillEntry
	for _, s := range out.Skills {
		if s.Experimental {
			experimental = append(experimental, s)
		} else {
			stable = append(stable, s)
		}
	}

	if len(out.Agents) > 0 {
		cmdio.LogString(ctx, "Plugin installs:")
		cmdio.LogString(ctx, "")
		var ab strings.Builder
		atw := tabwriter.NewWriter(&ab, 0, 4, 2, ' ', 0)
		fmt.Fprintln(atw, "  AGENT\tSTATUS")
		for _, a := range out.Agents {
			fmt.Fprintf(atw, "  %s\t%s\n", agentDisplayName(a.Name), agentStatusLabel(a, out.Release))
		}
		atw.Flush()
		cmdio.LogString(ctx, ab.String())
		cmdio.LogString(ctx, "")
	}

	cmdio.LogString(ctx, "Available raw skill directories ("+versionToken(out.Release)+"):")
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, renderSkillTable(stable, bothScopes))

	if len(experimental) > 0 {
		cmdio.LogString(ctx, "Experimental skills:")
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, renderSkillTable(experimental, bothScopes))
	}

	cmdio.LogString(ctx, summaryLine(out, scope))
}

// renderSkillTable formats a NAME/VERSION/INSTALLED table for a group of skills.
func renderSkillTable(skills []skillEntry, bothScopes bool) string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tVERSION\tINSTALLED")
	for _, s := range skills {
		fmt.Fprintf(tw, "  %s\tv%s\t%s\n", s.Name, s.LatestVersion, installedStatusFromEntry(s, bothScopes))
	}
	tw.Flush()
	return buf.String()
}

// agentStatusLabel renders the text-view status for an agent, collapsing the
// per-scope plugin records into a single line. A stale scope (version !=
// release) is surfaced over an up-to-date one so an outdated install is never
// hidden; project is preferred when every scope matches release.
func agentStatusLabel(a agentEntry, release string) string {
	version, upToDate := "", true
	for _, scope := range []string{installer.ScopeProject, installer.ScopeGlobal} {
		info, ok := a.Installed[scope]
		if !ok {
			continue
		}
		stale := info.Version != release
		if version == "" || (upToDate && stale) {
			version = info.Version
		}
		if stale {
			upToDate = false
		}
	}

	if upToDate {
		return "databricks plugin · " + versionToken(version) + " · up to date"
	}
	return "databricks plugin · " + versionToken(version) + " · update available"
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
			return fmt.Sprintf("%d/%d raw skill directories installed (global), %d/%d (project)", g.Installed, g.Total, p.Installed, p.Total)
		}
		if p.loaded {
			return fmt.Sprintf("%d/%d raw skill directories installed (project)", p.Installed, p.Total)
		}
		return fmt.Sprintf("%d/%d raw skill directories installed (global)", g.Installed, g.Total)
	case pOK:
		return fmt.Sprintf("%d/%d raw skill directories installed (project)", p.Installed, p.Total)
	default:
		return fmt.Sprintf("%d/%d raw skill directories installed (global)", g.Installed, g.Total)
	}
}
