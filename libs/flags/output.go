package flags

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Output controls how the CLI should produce its output.
type Output string

const (
	OutputText Output = "text"
	OutputJSON Output = "json"
)

func (f *Output) String() string {
	return string(*f)
}

func (f *Output) Set(s string) error {
	lower := strings.ToLower(s)
	switch lower {
	case `json`, `text`:
		*f = Output(lower)
	default:
		return errors.New("accepted arguments are json and text")
	}
	return nil
}

func (f *Output) Type() string {
	return "type"
}

// Complete is the Cobra compatible completion function for this flag.
func (f *Output) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		fmt.Sprint(OutputText),
		fmt.Sprint(OutputJSON),
	}, cobra.ShellCompDirectiveNoFileComp
}
