package aitools

import (
	"fmt"
	"sort"
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
	// --skills is accepted for forward-compat (future component types)
	// but currently skills is the only component, so the output is the same.
	var showSkills bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed AI tools components",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = showSkills
			return listSkillsFn(cmd)
		},
	}

	cmd.Flags().BoolVar(&showSkills, "skills", false, "Show detailed skills information")
	return cmd
}

func defaultListSkills(cmd *cobra.Command) error {
	ctx := cmd.Context()

	src := &installer.GitHubManifestSource{}
	latestTag, _, err := src.FetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	manifest, err := src.FetchManifest(ctx, latestTag)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}

	globalDir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return err
	}

	state, err := installer.LoadState(globalDir)
	if err != nil {
		log.Debugf(ctx, "Could not load install state: %v", err)
	}

	// Build sorted list of skill names.
	names := make([]string, 0, len(manifest.Skills))
	for name := range manifest.Skills {
		names = append(names, name)
	}
	sort.Strings(names)

	version := strings.TrimPrefix(latestTag, "v")
	cmdio.LogString(ctx, "Available skills (v"+version+"):")
	cmdio.LogString(ctx, "")

	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "  NAME\tVERSION\tINSTALLED")

	installedCount := 0
	for _, name := range names {
		meta := manifest.Skills[name]

		tag := ""
		if meta.Experimental {
			tag = " [experimental]"
		}

		installedStr := "not installed"
		if state != nil {
			if v, ok := state.Skills[name]; ok {
				installedCount++
				if v == meta.Version {
					installedStr = "v" + v + " (up to date)"
				} else {
					installedStr = "v" + v + " (update available)"
				}
			}
		}

		fmt.Fprintf(tw, "  %s%s\tv%s\t%s\n", name, tag, meta.Version, installedStr)
	}
	tw.Flush()
	cmdio.LogString(ctx, buf.String())

	cmdio.LogString(ctx, fmt.Sprintf("%d/%d skills installed (global)", installedCount, len(names)))
	return nil
}
