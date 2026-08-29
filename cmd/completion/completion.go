package completion

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	libcompletion "github.com/databricks/cli/libs/completion"
	"github.com/spf13/cobra"
)

// New returns the "completion" command with shell subcommands and
// install/uninstall/status subcommands. Providing a command named
// "completion" prevents Cobra's InitDefaultCompletionCmd from generating
// its default, so we replicate the shell subcommands here with runtime
// writer resolution to avoid the frozen-writer bug.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate the autocompletion script for the specified shell",
		Long: `Generate the autocompletion script for databricks for the specified shell.
See each sub-command's help for details on how to use the generated script.
`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              root.ReportUnknownSubcommand,
	}

	cmd.AddCommand(
		newBashCmd(),
		newZshCmd(),
		newFishCmd(),
		newPowershellCmd(),
		newInstallCmd(),
		newUninstallCmd(),
		newStatusCmd(),
	)

	return cmd
}

func newBashCmd() *cobra.Command {
	var noDesc bool
	cmd := &cobra.Command{
		Use:   "bash",
		Short: "Generate the autocompletion script for bash",
		Long: `Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(databricks completion bash)

To load completions for every new session, execute once:

#### Linux:

	databricks completion bash > /etc/bash_completion.d/databricks

#### macOS:

	databricks completion bash > $(brew --prefix)/etc/bash_completion.d/databricks

You will need to start a new shell for this setup to take effect.
`,
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletionV2(cmd.OutOrStdout(), !noDesc)
		},
	}
	cmd.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func newZshCmd() *cobra.Command {
	var noDesc bool
	cmd := &cobra.Command{
		Use:   "zsh",
		Short: "Generate the autocompletion script for zsh",
		Long: `Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(databricks completion zsh)

To load completions for every new session, execute once:

#### Linux:

	databricks completion zsh > "${fpath[1]}/_databricks"

#### macOS:

	databricks completion zsh > $(brew --prefix)/share/zsh/site-functions/_databricks

You will need to start a new shell for this setup to take effect.
`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDesc {
				return cmd.Root().GenZshCompletionNoDesc(cmd.OutOrStdout())
			}
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func newFishCmd() *cobra.Command {
	var noDesc bool
	cmd := &cobra.Command{
		Use:   "fish",
		Short: "Generate the autocompletion script for fish",
		Long: `Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	databricks completion fish | source

To load completions for every new session, execute once:

	databricks completion fish > ~/.config/fish/completions/databricks.fish

You will need to start a new shell for this setup to take effect.
`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), !noDesc)
		},
	}
	cmd.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

func newPowershellCmd() *cobra.Command {
	var noDesc bool
	cmd := &cobra.Command{
		Use:   "powershell",
		Short: "Generate the autocompletion script for powershell",
		Long: `Generate the autocompletion script for powershell.

To load completions in your current shell session:

	databricks completion powershell | Out-String | Invoke-Expression

To load completions for every new session, add the output of the above command
to your powershell profile.
`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDesc {
				return cmd.Root().GenPowerShellCompletion(cmd.OutOrStdout())
			}
			return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	return cmd
}

// warnIfCompinitMissing prints a warning when zsh completions are present but
// the user's .zshrc does not call compinit. Without compinit, neither our eval
// shim nor Homebrew's _databricks file will be loaded.
func warnIfCompinitMissing(ctx context.Context, shell libcompletion.Shell, home string) {
	if shell != libcompletion.Zsh {
		return
	}
	rcPath := libcompletion.TargetFilePath(shell, home)
	content, err := os.ReadFile(rcPath)
	if err != nil {
		return
	}
	if strings.Contains(string(content), "compinit") {
		return
	}
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Warning: zsh completions require the completion system to be initialized.")
	cmdio.LogString(ctx, "Add the following to your "+filepath.ToSlash(rcPath)+":")
	cmdio.LogString(ctx, "  autoload -U compinit && compinit")
}

// addShellFlag registers the --shell flag and its completion function on cmd.
func addShellFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "shell", "", "Shell type: bash, zsh, fish, powershell, powershell5")
	cmd.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		return []cobra.Completion{"bash", "zsh", "fish", "powershell", "powershell5"}, cobra.ShellCompDirectiveNoFileComp
	})
}
