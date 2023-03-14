package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Output controls how the CLI should produce its output.
type Output string

var (
	OutputText = Output("text")
	OutputJSON = Output("json")
)

func (f *Output) String() string {
	return string(*f)
}

func (f *Output) Set(s string) error {
	lower := strings.ToLower(s)
	switch lower {
	case OutputText.String():
		*f = Output(OutputText.String())
	case OutputJSON.String():
		*f = Output(OutputJSON.String())
	default:
		valid := []string{
			OutputText.String(),
			OutputJSON.String(),
		}
		return fmt.Errorf("accepted arguments are %s", strings.Join(valid, " and "))
	}
	return nil
}

func (f *Output) Type() string {
	return "type"
}

// Complete is the Cobra compatible completion function for this flag.
func (f *Output) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		OutputText.String(),
		OutputJSON.String(),
	}, cobra.ShellCompDirectiveNoFileComp
}
