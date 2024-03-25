package root

import (
	"fmt"

	"github.com/spf13/cobra"
)

type InvalidArgsError struct {
	// The command that was run.
	Command *cobra.Command
	// The error message.
	Message string
}

func (e *InvalidArgsError) Error() string {
	return fmt.Sprintf("%s\n\n%s", e.Message, e.Command.UsageString())
}

func ExactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) != n {
			return &InvalidArgsError{Message: fmt.Sprintf("accepts %d arg(s), received %d", n, len(args)), Command: cmd}
		}
		return nil
	}
}

func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
		return &InvalidArgsError{Message: msg, Command: cmd}
	}
	return nil
}

func MaximumNArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) > n {
			msg := fmt.Sprintf("accepts at most %d arg(s), received %d", n, len(args))
			return &InvalidArgsError{Message: msg, Command: cmd}
		}
		return nil
	}
}
