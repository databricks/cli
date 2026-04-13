package aitools

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/databricks/cli/experimental/aitools/lib/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

// listSkillsFn is the function used to render the skills list.
// It is a package-level var so tests can replace the data-fetching layer.
var listSkillsFn = defaultListSkills

func newListCmd() *cobra.Command {
	var projectFlag, globalFlag bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed AI tools components",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectFlag && globalFlag {
				return errors.New("cannot use --global and --project together")
			}
			// For list: no flag = show both scopes (empty string).
			var scope string
			if projectFlag {
				scope = installer.ScopeProject
			} else if globalFlag {
				scope = installer.ScopeGlobal
			}
			return listSkillsFn(cmd, scope)
		},
	}

	cmd.Flags().BoolVar(&projectFlag, "project", false, "Show only project-scoped skills")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Show only globally-scoped skills")
	return cmd
}

func defaultListSkills(cmd *cobra.Command, scope string) error {
	ctx := cmd.Context()

	ref := installer.GetSkillsRef(ctx)

	src := &installer.GitHubManifestSource{}
	manifest, err := src.FetchManifest(ctx, ref)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	// Load global state.
	var globalState *installer.InstallState
	if scope != installer.ScopeProject {
		globalDir, gErr := installer.GlobalSkillsDir(ctx)
		if gErr == nil {
			globalState, err = installer.LoadState(globalDir)
			if err != nil {
				log.Debugf(ctx, "Could not load global install state: %v", err)
			}
		}
	}

	// Load project state.
	var projectState *installer.InstallState
	if scope != installer.ScopeGlobal {
		projectDir, pErr := installer.ProjectSkillsDir(ctx)
		if pErr == nil {
			projectState, err = installer.LoadState(projectDir)
			if err != nil {
				log.Debugf(ctx, "Could not load project install state: %v", err)
			}
		}
	}

	// Build sorted list of skill names.
	names := make([]string, 0, len(manifest.Skills))
	for name := range manifest.Skills {
		names = append(names, name)
	}
	slices.Sort(names)

	version := strings.TrimPrefix(ref, "v")
	cmdio.LogString(ctx, "Available skills (v"+version+"):")
	cmdio.LogString(ctx, "")

	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tVERSION\tINSTALLED")

	bothScopes := globalState != nil && projectState != nil

	globalCount := 0
	projectCount := 0
	for _, name := range names {
		meta := manifest.Skills[name]

		tag := ""
		if meta.Experimental {
			tag = " [experimental]"
		}

		installedStr := installedStatus(name, meta.Version, globalState, projectState, bothScopes)
		if globalState != nil {
			if _, ok := globalState.Skills[name]; ok {
				globalCount++
			}
		}
		if projectState != nil {
			if _, ok := projectState.Skills[name]; ok {
				projectCount++
			}
		}

		fmt.Fprintf(tw, "  %s%s\tv%s\t%s\n", name, tag, meta.Version, installedStr)
	}
	tw.Flush()
	cmdio.LogString(ctx, buf.String())

	// Summary line.
	switch {
	case bothScopes:
		cmdio.LogString(ctx, fmt.Sprintf("%d/%d skills installed (global), %d/%d (project)", globalCount, len(names), projectCount, len(names)))
	case projectState != nil:
		cmdio.LogString(ctx, fmt.Sprintf("%d/%d skills installed (project)", projectCount, len(names)))
	case scope == installer.ScopeProject:
		cmdio.LogString(ctx, fmt.Sprintf("%d/%d skills installed (project)", 0, len(names)))
	default:
		cmdio.LogString(ctx, fmt.Sprintf("%d/%d skills installed (global)", globalCount, len(names)))
	}
	return nil
}

// installedStatus returns the display string for a skill's installation status.
func installedStatus(name, latestVersion string, globalState, projectState *installer.InstallState, bothScopes bool) string {
	globalVer := ""
	projectVer := ""

	if globalState != nil {
		globalVer = globalState.Skills[name]
	}
	if projectState != nil {
		projectVer = projectState.Skills[name]
	}

	if globalVer == "" && projectVer == "" {
		return "not installed"
	}

	// If both scopes have the skill, show the project version (takes precedence).
	if bothScopes && globalVer != "" && projectVer != "" {
		return versionLabel(projectVer, latestVersion) + " (project, global)"
	}

	if projectVer != "" {
		label := versionLabel(projectVer, latestVersion)
		if bothScopes {
			return label + " (project)"
		}
		return label
	}

	label := versionLabel(globalVer, latestVersion)
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
