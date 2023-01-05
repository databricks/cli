package bundle

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/repl"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/spf13/cobra"
)

func displayTable(w io.Writer, results *commands.Results) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(results)
}

var replCmd = &cobra.Command{
	Use: "repl [flags]",

	PreRunE: ConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// Ensure we have a command execution context.

		buf, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return err
		}

		r, err := repl.GetOrCreate(cmd.Context(), b, replLanguage)
		if err != nil {
			return err
		}

		out, err := r.Execute(cmd.Context(), buf)
		if err != nil {
			return err
		}

		if out.Status == commands.CommandStatusFinished {
			result := out.Results
			switch result.ResultType {
			case commands.ResultTypeError:
				fmt.Fprintf(cmd.OutOrStdout(), "Result: %#v", out.Results)
			case commands.ResultTypeImage:
				fmt.Fprintf(cmd.OutOrStdout(), "Result: %#v", out.Results)
			case commands.ResultTypeImages:
				fmt.Fprintf(cmd.OutOrStdout(), "Result: %#v", out.Results)
			case commands.ResultTypeTable:
				displayTable(cmd.OutOrStdout(), out.Results)

				// fmt.Fprintf(cmd.OutOrStdout(), "Result: %#v", out.Results)
			case commands.ResultTypeText:
				fmt.Fprintln(cmd.OutOrStdout(), result.Data)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Status: %#v", out)
		}

		return nil
	},
}

var replLanguage commands.Language

func init() {
	replCmd.Flags().VarP(&replLanguage, "language", "l", "Language (Python, SQL, R)")
	rootCmd.AddCommand(replCmd)
}
