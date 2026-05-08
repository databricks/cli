package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func validatePositive(n int) error {
	if n < 1 {
		return fmt.Errorf("--n must be at least 1, got %d", n)
	}
	return nil
}

func newSelectCmd() *cobra.Command {
	var n int
	cmd := &cobra.Command{
		Use:   "select",
		Short: "cmdio.Select (map; sorted alphabetically by name)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validatePositive(n)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			tuples := buildItems(n)
			items := make(map[string]string, len(tuples))
			for _, t := range tuples {
				items[t.Name] = t.Id
			}
			id, err := cmdio.Select(ctx, items, "Pick an item")
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, "Selected: "+id)
			return nil
		},
	}
	cmd.Flags().IntVar(&n, "n", 5, "number of items")
	return cmd
}

func newSelectOrderedCmd() *cobra.Command {
	var (
		n      int
		long   bool
		filter bool
	)
	cmd := &cobra.Command{
		Use:   "select-ordered",
		Short: "cmdio.SelectOrdered ([]Tuple; preserves insertion order)",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if long || filter {
				return nil
			}
			return validatePositive(n)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			var items []cmdio.Tuple
			switch {
			case filter:
				items = buildFilterItems()
			case long:
				items = buildLongItems()
			default:
				items = buildItems(n)
			}
			id, err := cmdio.SelectOrdered(ctx, items, "Pick an item")
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, "Selected: "+id)
			return nil
		},
	}
	cmd.Flags().IntVar(&n, "n", 5, "number of items (ignored with --long or --filter)")
	cmd.Flags().BoolVar(&long, "long", false, "use 8 items with 60+ char ids that overflow the terminal")
	cmd.Flags().BoolVar(&filter, "filter", false, "use 15 items with overlapping substrings (try typing 'al' or 'xyz')")
	cmd.MarkFlagsMutuallyExclusive("long", "filter")
	return cmd
}

func newRunSelectCmd() *cobra.Command {
	var (
		rich        bool
		conditional bool
	)
	cmd := &cobra.Command{
		Use:   "run-select",
		Short: "cmdio.RunSelect (custom SelectOptions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			switch {
			case rich:
				return runSelectRich(ctx)
			case conditional:
				return runSelectProfile(ctx)
			default:
				return runSelectPlain(ctx)
			}
		},
	}
	cmd.Flags().BoolVar(&rich, "rich", false, "use cluster-style rich Active/Inactive templates (bold + faint)")
	cmd.Flags().BoolVar(&conditional, "conditional", false, "use profile-style {{if .Host}} template branches and trailing meta-rows")
	cmd.MarkFlagsMutuallyExclusive("rich", "conditional")
	return cmd
}

func runSelectPlain(ctx context.Context) error {
	items := buildItems(5)
	i, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
		Label: "Pick an item",
		Items: items,
	})
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, "Selected: "+items[i].Id)
	return nil
}

func runSelectRich(ctx context.Context) error {
	items := buildClusterItems(ctx)
	i, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
		Label: "Choose a cluster",
		Items: items,
		Searcher: func(input string, idx int) bool {
			return strings.Contains(strings.ToLower(items[idx].Name), strings.ToLower(input))
		},
		StartInSearchMode: true,
		LabelTemplate:     `{{ . | faint }}`,
		Active:            `{{.Name | bold}} ({{.State}} {{.Access}} Runtime {{.Runtime}}) ({{.Id | faint}})`,
		Inactive:          `{{.Name}} ({{.State}} {{.Access}} Runtime {{.Runtime}})`,
		Selected:          `{{ "Selected cluster" | faint }}: {{ .Name | bold }} ({{ .Id | faint }})`,
	})
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, "Selected: "+items[i].Name+" ("+items[i].Id+")")
	return nil
}

func runSelectProfile(ctx context.Context) error {
	items := buildProfileItems()
	i, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
		Label:             "Select a profile",
		Items:             items,
		StartInSearchMode: true,
		Searcher: func(input string, idx int) bool {
			input = strings.ToLower(input)
			return strings.Contains(strings.ToLower(items[idx].Name), input) ||
				strings.Contains(strings.ToLower(items[idx].Host), input)
		},
		LabelTemplate: `{{ . | faint }}`,
		Active:        `{{.Name | bold}}{{if .Host}} ({{.Host | faint}}){{end}}`,
		Inactive:      `{{.Name}}{{if .Host}} ({{.Host}}){{end}}`,
		Selected:      `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
	})
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, "Selected: "+items[i].Name)
	return nil
}
