package aitools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// NewMcpCmd returns the `aitools mcp` group. It stays hidden until the whole MCP
// stack has landed, so a release never ships `add` without `serve`.
func NewMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Manage Databricks managed MCP servers for coding agents",
		Long: `Databricks managed MCP servers give coding agents tools for Databricks services, such as GitHub or web search.
Use 'add' to install them into coding agents, bound to a Databricks profile, 'list' to see what is installed,
and 'remove' to uninstall them. Some services use your account with an external provider, such as GitHub:
'login' signs you in to one and 'logout' removes the sign-in.`,
		Hidden: true,
	}
	cmd.AddCommand(NewMcpAddCmd())
	cmd.AddCommand(NewMcpListCmd())
	cmd.AddCommand(NewMcpRemoveCmd())
	cmd.AddCommand(NewMcpLoginCmd())
	cmd.AddCommand(NewMcpLogoutCmd())
	cmd.AddCommand(NewMcpServeCmd())
	return cmd
}

// lockMCP takes the lock that add and remove hold from before they read the install records until they have saved them,
// so that one of them never plans or writes while another does. It fails while another one holds the lock rather than
// waiting, as uninstall does for an automatic installation: the other run may be waiting at a prompt.
func lockMCP(ctx context.Context) (unlock func(), err error) {
	unlock, err = autoinstall.LockMCP(ctx)
	if err != nil {
		return nil, fmt.Errorf("another databricks aitools mcp add or remove is running; run this command again when it finishes: %w", err)
	}
	// add releases the lock before its sign-in, and again when it returns.
	return sync.OnceFunc(unlock), nil
}

// failure returns the message shown to the user, tagged with the category reported in JSON.
func failure(category protos.AitoolsErrorCategory, format string, args ...any) error {
	return withCategory(fmt.Errorf(format, args...), category)
}

// withAgentsHint names --agents as the way out of a read-only agent config file. The writer does not know the
// flag, so the command that takes --agents adds the hint. Other errors are returned as they are.
func withAgentsHint(err error, agent string) error {
	ro, ok := errors.AsType[*agentconfig.ReadOnlyError](err)
	if !ok {
		return err
	}
	return fmt.Errorf("%s is read-only; make it writable or leave %s out of --agents: %w", ro.Path, agent, ro.Err)
}

// failureWithCause is failure for a message that does not repeat its cause. cause is kept for errors.Is and errors.As
// but is not printed.
func failureWithCause(category protos.AitoolsErrorCategory, cause error, format string, args ...any) error {
	return withCategory(&displayError{msg: fmt.Sprintf(format, args...), err: cause}, category)
}

// errorCategory classifies an error by its typed errors only, never by its text. Failures without a
// category of their own fall through to the install classification, as in the other aitools commands.
func errorCategory(err error) protos.AitoolsErrorCategory {
	if ce, ok := errors.AsType[*categorizedError](err); ok {
		return ce.category
	}
	// An uncategorized binding failure falls through, so that its cause (a filesystem or network error) is classified.
	if be, ok := errors.AsType[*mcp.BindError](err); ok && be.Category != mcp.BindUncategorized {
		return protos.AitoolsErrorCategory(be.Category)
	}
	switch {
	case errors.Is(err, agentconfig.ErrConfigInvalid):
		return protos.AitoolsErrorCategoryAgentConfigInvalid
	case errors.Is(err, agentconfig.ErrNotOwned):
		return protos.AitoolsErrorCategoryMCPNameConflict
	}
	return classifyInstallError(err)
}

// resolveProfile picks the profile a non-interactive command binds to: the
// --profile flag, then DATABRICKS_CONFIG_PROFILE, then the default profile, then the only
// workspace profile. It never prompts.
func resolveProfile(ctx context.Context, flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if name := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); name != "" {
		return name, nil
	}
	if name := databrickscfg.ResolveDefaultProfile(ctx); name != "" {
		return name, nil
	}

	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return "", err
	}
	profiles, err := profiler.LoadProfiles(ctx, mcpWorkspaceProfile)
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return "", err
	}
	switch len(profiles) {
	case 0:
		return "", failure(protos.AitoolsErrorCategoryNoProfile, "no workspace profiles in %s. Run: databricks auth login --host <url> --profile <name>", filepath.ToSlash(path))
	case 1:
		return profiles[0].Name, nil
	}
	return "", failureWithCause(protos.AitoolsErrorCategoryProfileRequired, &profileChoice{profiles: profiles}, "several workspace profiles exist; pass --profile. Found: %s", strings.Join(profiles.Names(), ", "))
}

// mcpWorkspaceProfile reports whether auto-detect may pick p. It accepts the profiles that --profile accepts: a
// workspace profile, or a profile whose host names its workspace in ?w=, which is an account profile otherwise.
func mcpWorkspaceProfile(p profile.Profile) bool {
	return profile.MatchWorkspaceProfiles(p) || auth.ExtractHostQueryParams(p.Host).WorkspaceID != ""
}

// profileFlag returns the root --profile value, or "" when it is not set.
func profileFlag(cmd *cobra.Command) string {
	return cmd.Flag("profile").Value.String()
}
