package tui

import "github.com/spf13/cobra"

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Test terminal UI components (spinners, prompts, etc.)",
	}

	cmd.AddCommand(
		newAskCmd(),
		newAskSelectCmd(),
		newAskYesOrNoCmd(),
		newColorsCmd(),
		newPromptCmd(),
		newRunSelectCmd(),
		newSecretCmd(),
		newSelectCmd(),
		newSelectOrderedCmd(),
		newSpinnerCmd(),
	)

	return cmd
}
