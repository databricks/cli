package sandbox

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	sshtunnel "github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func newSSHSetupCommand() *cobra.Command {
	var alias, sshConfig, releasesDir string
	var overwrite bool
	cmd := &cobra.Command{
		Use: "ssh-setup [sandbox-id]", Short: "Configure native SSH tools for a sandbox",
		Long: `Create a local SSH Host entry for native ssh, scp, sftp, and tools that use SSH.

The CLI creates or reuses a local key. On each connection, the generated
ProxyCommand prepares the sandbox and opens authenticated WebSocket port forwarding.
Pass --profile with a named Databricks profile so later connections select the same workspace.
Databricks CLI and that profile's credentials must remain available locally.

Examples:
  databricks sandbox ssh-setup happy-panda-1234 --name my-sandbox --profile my-workspace
  ssh my-sandbox
  scp ./example.txt my-sandbox:/tmp/example.txt

Interactive 'databricks sandbox ssh' works without this setup step.`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
			return root.MustWorkspaceClient(cmd, args)
		},
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := sshtunnel.ValidateHostAlias(alias); err != nil {
				return err
			}
			ctx := cmd.Context()
			profileFlag := cmd.Flag("profile")
			if profileFlag == nil || !profileFlag.Changed || profileFlag.Value.String() == "" {
				return errors.New("ssh-setup requires a named Databricks profile; run 'databricks auth login --profile NAME' and rerun with --profile NAME")
			}
			w := cmdctx.WorkspaceClient(ctx)
			requested := ""
			if len(args) == 1 {
				requested = args[0]
			}
			name, err := resolveSSHName(ctx, w, requested)
			if err != nil {
				return err
			}
			identity, err := prepareSSHIdentity(ctx, w, name)
			if err != nil {
				return err
			}
			executable, err := os.Executable()
			if err != nil {
				return err
			}
			options := sshProxyOptions{Profile: w.Config.Profile, ReleasesDir: releasesDir, IdentityScope: identity.HostKeyAlias}
			if w.Config.ConfigFile != "" {
				file, err := config.LoadFile(w.Config.ConfigFile)
				if err != nil {
					return fmt.Errorf("load SSH profile configuration: %w", err)
				}
				options.ConfigFile, err = filepath.Abs(file.Path())
				if err != nil {
					return fmt.Errorf("resolve SSH profile configuration path: %w", err)
				}
				options.ConfigFile = filepath.ToSlash(options.ConfigFile)
			}
			proxy, err := sshProxyCommand(executable, name, options)
			if err != nil {
				return err
			}
			location, err := sshtunnel.WriteHostConfig(ctx, alias, sshConfig, proxy, identity, overwrite)
			if err != nil {
				return err
			}
			cmdio.LogString(ctx, fmt.Sprintf("Created SSH configuration at %s. Connect with 'ssh %s' or 'scp ./file %s:/tmp/file'.", strings.ReplaceAll(location, `\`, "/"), alias, alias))
			return nil
		},
	}
	cmd.Flags().StringVar(&alias, "name", "", "Local SSH host alias")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&sshConfig, "ssh-config", "", "Path to SSH config file (default ~/.ssh/config)")
	cmd.Flags().StringVar(&releasesDir, "releases-dir", "", "Directory containing version-matched Linux CLI archives")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Replace an existing managed SSH host entry")
	return cmd
}
