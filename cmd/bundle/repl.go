package bundle

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/chzyer/readline"
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/repl"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/spf13/cobra"
)

func displayTable(w io.Writer, results *commands.Results) error {
	out, err := repl.TableToMap(results)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(out)
	return nil
}

var replCmd = &cobra.Command{
	Use: "repl [flags]",

	PreRunE: ConfigureBundle,
	RunE: func(cmd *cobra.Command, args []string) error {
		b := bundle.Get(cmd.Context())

		// Ensure we have a command execution context.
		r, err := repl.GetOrCreate(cmd.Context(), b, replLanguage)
		if err != nil {
			return err
		}

		rl, err := readline.New("> ")
		if err != nil {
			panic(err)
		}

		defer rl.Close()

		for {
			line, err := rl.Readline()
			if err != nil { // io.EOF
				break
			}

			if line == "" {
				continue
			}

			out, err := r.Execute(cmd.Context(), []byte(line))
			if err != nil {
				return err
			}

			if out.Status == commands.CommandStatusFinished {
				result := out.Results
				switch result.ResultType {
				case commands.ResultTypeError:
					fmt.Fprintf(cmd.OutOrStdout(), out.Results.Summary)
				case commands.ResultTypeImage:
					fmt.Fprintf(cmd.OutOrStdout(), "Result: %#v", out.Results)
				case commands.ResultTypeImages:
					fmt.Fprintf(cmd.OutOrStdout(), "Result: %#v", out.Results)
				case commands.ResultTypeTable:
					err = displayTable(cmd.OutOrStdout(), out.Results)
					if err != nil {
						return err
					}

				case commands.ResultTypeText:
					fmt.Fprintln(cmd.OutOrStdout(), result.Data)
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Status: %#v", out)
			}
		}

		return nil
	},
}

var replLanguage commands.Language

func init() {
	replCmd.Flags().VarP(&replLanguage, "language", "l", "Language (Python, SQL, R)")
	rootCmd.AddCommand(replCmd)
}
