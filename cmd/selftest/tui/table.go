package tui

import (
	"fmt"

	"github.com/databricks/cli/libs/tableview"
	"github.com/spf13/cobra"
)

func newTableCmd() *cobra.Command {
	var (
		rows int
		cols int
	)
	cmd := &cobra.Command{
		Use:   "table",
		Short: "tableview.Run over generated rows (interactive browser)",
		Args:  cobra.NoArgs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if rows < 0 {
				return fmt.Errorf("--rows must be non-negative, got %d", rows)
			}
			if cols < 1 {
				return fmt.Errorf("--cols must be at least 1, got %d", cols)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			columns := make([]string, cols)
			for c := range cols {
				columns[c] = fmt.Sprintf("col-%02d", c+1)
			}
			data := make([][]string, rows)
			for r := range rows {
				data[r] = make([]string, cols)
				for c := range cols {
					data[r][c] = fmt.Sprintf("r%04d-c%02d", r+1, c+1)
				}
			}
			// tableview.Run does not report how the user exited, only the error.
			err := tableview.Run(cmd.Context(), cmd.OutOrStdout(), columns, data)
			fmt.Fprintf(cmd.ErrOrStderr(), "tableview.Run returned: %v\n", err)
			return err
		},
	}
	cmd.Flags().IntVar(&rows, "rows", 100, "number of generated rows")
	cmd.Flags().IntVar(&cols, "cols", 4, "number of generated columns")
	return cmd
}
