package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newPromptCmd() *cobra.Command {
	var (
		mask     bool
		validate bool
	)
	cmd := &cobra.Command{
		Use:   "prompt",
		Short: "cmdio.RunPrompt (single-line text input)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			opts := cmdio.PromptOptions{
				Label: "Enter a value",
			}
			if mask {
				opts.Mask = '*'
			}
			if validate {
				opts.Validate = func(input string) error {
					if !strings.Contains(input, "://") {
						return errors.New("value must contain '://'")
					}
					return nil
				}
			}
			value, err := cmdio.RunPrompt(ctx, opts)
			if err != nil {
				return err
			}
			if mask {
				cmdio.LogString(ctx, fmt.Sprintf("Entered %d characters", len(value)))
				return nil
			}
			cmdio.LogString(ctx, "Entered: "+value)
			return nil
		},
	}
	cmd.Flags().BoolVar(&mask, "mask", false, "echo input as '*'")
	cmd.Flags().BoolVar(&validate, "validate", false, "require '://' in input")
	return cmd
}

func newSecretCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "secret",
		Short: "cmdio.Secret (masked password input)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			value, err := cmdio.Secret(ctx, "Personal access token")
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, fmt.Sprintf("Entered %d characters", len(value)))
			return nil
		},
	}
}
