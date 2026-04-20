package ucm

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"text/tabwriter"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/cmd/ucm/utils"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/spf13/cobra"
)

// tfstateEnvelope is the minimal shape we need out of terraform.tfstate to
// produce a resource-count summary. Forked (not imported) from the terraform
// project's Go API so ucm doesn't pin on an internal schema.
type tfstateEnvelope struct {
	Resources []struct {
		Type string `json:"type"`
	} `json:"resources"`
}

func newSummaryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Summarize deployed resources by type.",
		Long: `Summarize the resources currently tracked by the ucm deploy state.

Reads the local terraform state cached under .databricks/ucm/<target>/ and
prints a table of resource type + count. Run ` + "`ucm deploy`" + ` (or at least
` + "`ucm plan`" + `) first; without a local state the table is empty.`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		u := utils.ProcessUcm(cmd, utils.ProcessOptions{})
		ctx := cmd.Context()
		if u == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		statePath := filepath.Join(deploy.LocalStateDir(u), deploy.TfStateFileName)
		counts, err := readTfstateCounts(statePath)
		if err != nil {
			return fmt.Errorf("read local state %s: %w", filepath.ToSlash(statePath), err)
		}

		out := cmd.OutOrStdout()
		if len(counts) == 0 {
			fmt.Fprintln(out, "No deployed resources found. Run `ucm deploy` first.")
			return nil
		}

		types := make([]string, 0, len(counts))
		for t := range counts {
			types = append(types, t)
		}
		sort.Strings(types)

		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "TYPE\tCOUNT")
		for _, t := range types {
			fmt.Fprintf(tw, "%s\t%d\n", t, counts[t])
		}
		return tw.Flush()
	}

	return cmd
}

// readTfstateCounts opens the terraform.tfstate at path and returns a map of
// resource type → count. A missing state file is treated as "no resources"
// rather than an error so the first-run / pre-deploy path stays clean.
func readTfstateCounts(path string) (map[string]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var env tfstateEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse tfstate: %w", err)
	}

	counts := make(map[string]int, len(env.Resources))
	for _, r := range env.Resources {
		counts[r.Type]++
	}
	return counts, nil
}
