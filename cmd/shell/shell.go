package shell

import (
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Launches interactive CLI prompt",
	// TODO: improve context/client sharing between commands launched in this shell
	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		ctx = useragent.InContext(ctx, "feature", "shell")
		root.RootCmd.SetContext(ctx)
		root.RootCmd.AddCommand(&cobra.Command{
			Use:   "exit",
			Short: "Exit prompt",
			Run: func(cmd *cobra.Command, args []string) {
				os.Exit(0)
			},
		})
		p := prompt.New(func(in string) {
			promptArgs := strings.Fields(in)
			os.Args = append([]string{os.Args[0]}, promptArgs...)
			err = root.RootCmd.Execute()
			if err != nil {
				cmd.PrintErrln(err)
			}
		}, suggester(root.RootCmd),
			prompt.OptionPrefix("🧱 > "),
			prompt.OptionTitle("bricks shell"),
			prompt.OptionCompletionOnDown(),
			prompt.OptionShowCompletionAtStart(),
			prompt.OptionDescriptionBGColor(prompt.White),
			prompt.OptionSuggestionTextColor(prompt.Black),
			prompt.OptionSuggestionBGColor(prompt.White),
			prompt.OptionSelectedSuggestionBGColor(prompt.LightGray),
			prompt.OptionSelectedDescriptionBGColor(prompt.LightGray),
			prompt.OptionMaxSuggestion(10))
		p.Run()
		return nil
	},
}

// we could also add dynamic argument completion, which is reasonable performance-wise
// only in the interactive shell, as ValidArgsFunction gets called every time bash/zsh/psh
// types a character, while every call is isolated from another. filesystem cache is another
// option.
//
// See:
// https://github.com/c-bata/gh-prompt/blob/master/completer/argument.go#L28-L51
// https://github.com/c-bata/gh-prompt/blob/master/completer/client.go
func suggester(root *cobra.Command) func(d prompt.Document) (sgg []prompt.Suggest) {
	return func(d prompt.Document) (sgg []prompt.Suggest) {
		args := strings.Fields(d.CurrentLine())
		cmd, _, _ := root.Find(args)
		// if err != nil {
		// 	return nil
		// }
		wordBeforeCursor := d.GetWordBeforeCursor()
		if cmd.ValidArgsFunction != nil {
			cmd.SetContext(root.Context())
			valid, _ := cmd.ValidArgsFunction(cmd, args, wordBeforeCursor)
			for _, v := range valid {
				sgg = append(sgg, prompt.Suggest{
					Text: v,
				})
			}
		}
		cmd.LocalFlags().VisitAll(func(flag *pflag.Flag) {
			sgg = append(sgg, prompt.Suggest{
				Text:        "--" + flag.Name,
				Description: flag.Usage,
			})
		})
		for _, c := range cmd.Commands() {
			sgg = append(sgg, prompt.Suggest{
				Text:        c.Name(),
				Description: c.Short,
			})
		}
		return prompt.FilterHasPrefix(sgg, wordBeforeCursor, true)
	}
}

func init() {
	root.RootCmd.AddCommand(shellCmd)
}
