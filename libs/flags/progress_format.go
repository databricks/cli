package flags

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type ProgressLogFormat string

var (
	ModeAppend  = ProgressLogFormat("append")
	ModeInplace = ProgressLogFormat("inplace")
	ModeJson    = ProgressLogFormat("json")
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
	case ModeInplace.String():
		*p = ProgressLogFormat(ModeInplace.String())
	case ModeJson.String():
		*p = ProgressLogFormat(ModeJson.String())
	case ModeDefault.String():
		// We include ModeDefault here for symmetry reasons so this flag value
		// can be unset after test runs. We should not point this value in error
		// messages though since it's internal only
		*p = ProgressLogFormat(ModeJson.String())
	default:
		valid := []string{
			ModeAppend.String(),
			ModeInplace.String(),
			ModeJson.String(),
		}
		return fmt.Errorf("accepted arguments are [%s]", strings.Join(valid, ", "))
	}
	return nil
}

func (p *ProgressLogFormat) Type() string {
	return "format"
}

// Complete is the Cobra compatible completion function for this flag.
func (f *ProgressLogFormat) Complete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return []string{
		"append",
		"inplace",
		"json",
	}, cobra.ShellCompDirectiveNoFileComp
}
