package aircmd

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// renderBundlesText prints the bundle list as a table (job id, name, user,
// created), or a friendly line when empty.
func renderBundlesText(cmd *cobra.Command, bundles []airBundle) {
	out := cmd.OutOrStdout()
	if len(bundles) == 0 {
		fmt.Fprintln(out, "No bundles found.")
		return
	}
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Job ID\tName\tUser\tCreated")
	for _, b := range bundles {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", b.JobID, b.Name, orNA(b.User), bundleCreated(b))
	}
	tw.Flush()
}

// renderBundleDetail prints a single bundle's metadata plus its recent runs.
func renderBundleDetail(cmd *cobra.Command, data getBundleData) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Bundle:   %s\n", data.Name)
	fmt.Fprintf(out, "Job ID:   %s\n", data.JobID)
	fmt.Fprintf(out, "User:     %s\n", orNA(data.User))
	fmt.Fprintf(out, "Job URL:  %s\n", data.DashboardURL)

	if len(data.Runs) == 0 {
		fmt.Fprintln(out, "\nNo runs yet.")
		return
	}
	fmt.Fprintln(out, "\nRuns:")
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Run ID\tStatus\tStarted\tDuration")
	for _, r := range data.Runs {
		started := na
		if r.StartedAt != nil {
			started = *r.StartedAt
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", r.RunID, r.Status, started, orNA(r.Duration))
	}
	tw.Flush()
}

// bundleCreated formats a bundle's creation time, or "N/A" when unset.
func bundleCreated(b airBundle) string {
	if b.Created == nil {
		return na
	}
	return strings.TrimSpace(isoFormat(time.UnixMilli(*b.Created)))
}
