package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type ProgressLogFormat string

var (
	ModeAppend  = ProgressLogFormat("append")
	ModeDefault = ProgressLogFormat("default")
)

func (p *ProgressLogFormat) String() string {
	return string(*p)
}

func NewProgressLogFormat() ProgressLogFormat {
	return ModeDefault
}

func (p *ProgressLogFormat) Set(s string) error {
	lower := strings.ToLower(s)
	switch lower {
	case ModeAppend.String():
		*p = ProgressLogFormat(ModeAppend.String())
	case ModeDefault.String():
		// We include ModeDefault here for symmetry reasons so this flag value
		// can be unset after test runs. We should not point this value in error
		// messages though since it's internal only
		*p = ProgressLogFormat(ModeAppend.String())
	default:
		return fmt.Errorf("accepted arguments are [%s]", ModeAppend.String())
	}
	return nil
}

func (p *ProgressLogFormat) Type() string {
	return "format"
}

// Complete is the Cobra compatible completion function for this flag.
func (p *ProgressLogFormat) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"append",
	}, cobra.ShellCompDirectiveNoFileComp
}
