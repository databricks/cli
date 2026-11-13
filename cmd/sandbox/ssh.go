package sandbox

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"unicode"

	"github.com/databricks/cli/cmd/root"
	sshtunnel "github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/execv"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/spf13/cobra"
)

func newSSHCommand() *cobra.Command {
	var proxy bool
	var releasesDir string
	var expectedIdentity string
	var configFile string
	cmd := &cobra.Command{
		Use:   "ssh [sandbox-id] [-- <ssh-args-or-command>...]",
		Short: "SSH into a Sandbox environment",
		Long: `Connect to your default or a named sandbox through authenticated WebSocket port forwarding.

The CLI creates a local SSH key automatically. No Gateway SSH key registration
or persistent SSH Host configuration is required. The sandbox must have Python 3
and openssh-server installed, and the local machine must have OpenSSH.

Examples:
  databricks sandbox ssh
  databricks sandbox ssh happy-panda-1234
  databricks sandbox ssh happy-panda-1234 -- cat /etc/os-release
  databricks sandbox ssh -- -L 8080:localhost:8080`,
		Args: cobra.ArbitraryArgs, ValidArgsFunction: completeSandboxIDs,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if configFile == "" {
				return root.MustWorkspaceClient(cmd, args)
			}
			if !proxy {
				return errors.New("--ssh-config-file requires --proxy")
			}
			// The SDK reads this setting from process environment during initialization.
			previous, present := env.Lookup(cmd.Context(), "DATABRICKS_CONFIG_FILE")
			if err := os.Setenv("DATABRICKS_CONFIG_FILE", configFile); err != nil {
				return err
			}
			defer func() {
				if present {
					_ = os.Setenv("DATABRICKS_CONFIG_FILE", previous)
				} else {
					_ = os.Unsetenv("DATABRICKS_CONFIG_FILE")
				}
			}()
			return root.MustWorkspaceClient(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var requested string
			var extra []string
			dash := cmd.ArgsLenAtDash()
			if dash == -1 {
				dash = len(args)
			}
			if dash > 1 {
				return errors.New("specify at most one sandbox before --")
			}
			if dash == 1 {
				requested = args[0]
			}
			extra = args[dash:]
			if proxy && len(extra) != 0 {
				return errors.New("--proxy does not accept SSH arguments")
			}
			ctx := cmd.Context()
			if proxy {
				var stop context.CancelFunc
				ctx, stop = signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
				defer stop()
			}
			w := cmdctx.WorkspaceClient(ctx)
			name, err := resolveSSHName(ctx, w, requested)
			if err != nil {
				return err
			}
			identity, err := prepareSSHIdentity(ctx, w, name)
			if err != nil {
				return err
			}
			if expectedIdentity != "" && identity.HostKeyAlias != expectedIdentity {
				return errors.New("SSH workspace or caller changed; rerun ssh-setup with the intended Databricks profile")
			}
			if proxy {
				return sshtunnel.ConnectProxy(ctx, w, name, identity, releasesDir, os.Stdin, cmd.OutOrStdout(), cmd.ErrOrStderr())
			}
			options, err := sshClientOptions(cmd, w.Config, name, identity, releasesDir, extra)
			if err != nil {
				return err
			}
			return execv.Execv(options)
		},
	}
	cmd.Flags().BoolVar(&proxy, "proxy", false, "Relay SSH bytes over stdin/stdout without launching ssh")
	cmd.Flags().MarkHidden("proxy")
	cmd.Flags().StringVar(&expectedIdentity, "ssh-identity", "", "Expected local SSH identity scope")
	cmd.Flags().MarkHidden("ssh-identity")
	cmd.Flags().StringVar(&configFile, "ssh-config-file", "", "Databricks configuration file saved by ssh-setup")
	cmd.Flags().MarkHidden("ssh-config-file")
	cmd.Flags().StringVar(&releasesDir, "releases-dir", "", "Directory containing version-matched Linux CLI archives")
	return cmd
}

func sshClientOptions(cmd *cobra.Command, cfg *config.Config, name string, identity *sshtunnel.Identity, releasesDir string, extra []string) (execv.Options, error) {
	executable, err := os.Executable()
	if err != nil {
		return execv.Options{}, err
	}
	profile := ""
	if flag := cmd.Flag("profile"); flag != nil && flag.Changed {
		profile = flag.Value.String()
	}
	target := ""
	for _, flagName := range []string{"target", "environment"} {
		if flag := cmd.Flag(flagName); flag != nil && flag.Value.String() != "" {
			target = flag.Value.String()
			break
		}
	}
	transport, err := sshProxyCommand(executable, name, sshProxyOptions{Profile: profile, ReleasesDir: releasesDir, IdentityScope: identity.HostKeyAlias, Target: target})
	if err != nil {
		return execv.Options{}, err
	}
	return execv.Options{Args: buildSSHArgs(identity, transport, extra), Env: sshProcessEnvironment(cmd.Context(), cfg)}, nil
}

func sshProcessEnvironment(ctx context.Context, cfg *config.Config) []string {
	variables := env.All(ctx)
	variables["DATABRICKS_HOST"] = cfg.Host
	variables["DATABRICKS_ACCOUNT_ID"] = cfg.AccountID
	variables["DATABRICKS_WORKSPACE_ID"] = cfg.WorkspaceID
	if cfg.Profile != "" {
		// An implicit profile must retain environment-first auth precedence.
		// Converting it to --profile would give the profile precedence instead.
		variables["DATABRICKS_CONFIG_PROFILE"] = cfg.Profile
	}
	if cfg.ConfigFile != "" {
		variables["DATABRICKS_CONFIG_FILE"] = cfg.ConfigFile
	}
	result := make([]string, 0, len(variables))
	for key, value := range variables {
		result = append(result, key+"="+value)
	}
	return result
}

func resolveSSHName(ctx context.Context, w *databricks.WorkspaceClient, requested string) (string, error) {
	profile := w.Config.Profile
	if profile == "" {
		profile = w.Config.Host
	}
	savedDefault := requested == ""
	if savedDefault {
		requested = getDefault(ctx, profile)
	}
	if requested == "" {
		return "", errors.New("no default sandbox configured — specify a sandbox ID or run 'databricks sandbox default <id>'")
	}
	id, err := resolveLocalID(ctx, profile, strings.TrimPrefix(requested, "sandboxes/"))
	if err != nil {
		return "", err
	}
	if id == "" || strings.ContainsAny(id, "/\\?#") || strings.ContainsFunc(id, unicode.IsSpace) {
		return "", fmt.Errorf("invalid sandbox ID %q", id)
	}
	name := "sandboxes/" + id
	if _, err := w.Sandbox.GetSandbox(ctx, sdk.GetSandboxRequest{Name: name}); err != nil {
		if savedDefault && errors.Is(err, apierr.ErrNotFound) {
			return "", fmt.Errorf("saved default %q is unavailable through the Sandbox API; verify the profile or choose another sandbox: %w", id, err)
		}
		return "", err
	}
	return name, nil
}

func prepareSSHIdentity(ctx context.Context, w *databricks.WorkspaceClient, name string) (*sshtunnel.Identity, error) {
	caller, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return nil, fmt.Errorf("resolve SSH caller: %w", err)
	}
	workspace, err := auth.ResolveWorkspaceID(ctx, w)
	if err != nil {
		return nil, fmt.Errorf("resolve SSH workspace: %w", err)
	}
	return sshtunnel.LocalIdentity(ctx, w.Config.Host+"/"+workspace, caller.Id, name)
}

type sshProxyOptions struct {
	Profile, ReleasesDir, IdentityScope, Target, ConfigFile string
}

func sshProxyCommand(executable, name string, options sshProxyOptions) (string, error) {
	args := []string{executable, "sandbox", "ssh", name, "--proxy"}
	if options.Profile != "" {
		args = append(args, "--profile", options.Profile)
	}
	if options.ReleasesDir != "" {
		args = append(args, "--releases-dir", options.ReleasesDir)
	}
	if options.IdentityScope != "" {
		args = append(args, "--ssh-identity", options.IdentityScope)
	}
	if options.Target != "" {
		args = append(args, "--target", options.Target)
	}
	if options.ConfigFile != "" {
		args = append(args, "--ssh-config-file", options.ConfigFile)
	}
	for i, arg := range args {
		// OpenSSH expands percent tokens before invoking the shell, even inside quotes.
		if strings.ContainsFunc(arg, unicode.IsControl) {
			return "", errors.New("control characters are not supported in SSH proxy arguments")
		}
		arg = strings.ReplaceAll(arg, "%", "%%")
		if runtime.GOOS == "windows" {
			quoted, ok := shellquote.WindowsArg(arg)
			if !ok {
				return "", fmt.Errorf("unsupported Windows SSH proxy argument %q", arg)
			}
			args[i] = quoted
		} else {
			args[i] = shellquote.BashArg(arg)
		}
	}
	return strings.Join(args, " "), nil
}

func buildSSHArgs(identity *sshtunnel.Identity, proxyCommand string, extraArgs []string) []string {
	args := []string{
		"ssh", "-i", identity.KeyPath, "-o", "IdentitiesOnly=yes", "-o", "PreferredAuthentications=publickey",
		// Include authentication and the proxy's ten-minute remote setup budget.
		"-o", "ConnectTimeout=900",
		"-o", "StrictHostKeyChecking=yes", "-o", "UserKnownHostsFile=" + quoteSSHOption(identity.KnownHostsPath),
		"-o", "HostKeyAlias=" + identity.HostKeyAlias, "-o", "ServerAliveInterval=30", "-o", "ProxyCommand=" + proxyCommand,
		"sandbox-agent@" + identity.HostKeyAlias,
	}
	if len(extraArgs) == 1 {
		return append(args, extraArgs[0])
	}
	for _, arg := range extraArgs {
		args = append(args, shellquote.BashArg(arg))
	}
	return args
}

func quoteSSHOption(value string) string {
	return `"` + strings.ReplaceAll(filepathToSSH(value), `"`, `\"`) + `"`
}

func filepathToSSH(value string) string { return strings.ReplaceAll(value, `\`, "/") }
