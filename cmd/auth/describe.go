package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

var authTemplate = `{{"Host:" | bold}} {{.Status.Details.Host}}
{{- if .Status.AccountID}}
{{"Account ID:" | bold}} {{.Status.AccountID}}
{{- end}}
{{- if .Status.Username}}
{{"User:" | bold}} {{.Status.Username}}
{{- end}}
{{"Authenticated with:" | bold}} {{.Status.Details.AuthType}}
{{- if .Status.TokenStorage}}
{{"Token storage:" | bold}} {{.Status.TokenStorage.Mode}}, {{.Status.TokenStorage.Location}} {{ printf "(from %s)" .Status.TokenStorage.Source | italic}}
{{- end}}
-----
` + configurationTemplate

var errorTemplate = `Unable to authenticate: {{.Status.Error}}
{{- if .Status.TokenStorage}}
{{"Token storage:" | bold}} {{.Status.TokenStorage.Mode}}, {{.Status.TokenStorage.Location}} {{ printf "(from %s)" .Status.TokenStorage.Source | italic}}
{{- end}}
-----
` + configurationTemplate

const configurationTemplate = `Current configuration:
  {{- $details := .Status.Details}}
  {{- range $a := .ConfigAttributes}}
  {{- $k := $a.Name}}
  {{- if index $details.Configuration $k}}
  {{- $v := index $details.Configuration $k}}
  {{if $v.AuthTypeMismatch}}~{{else}}✓{{end}} {{$k | bold}}: {{$v.Value}}
  {{- if not (eq $v.Source.String "dynamic configuration")}}
  {{- " (from" | italic}} {{$v.Source.String | italic}}
  {{- if $v.AuthTypeMismatch}}, {{ "not used for auth type " | red | italic }}{{$details.AuthType | red | italic}}{{end}})
  {{- end}}
  {{- end}}
  {{- end}}
`

func newDescribeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describes the credentials and the source of those credentials, being used by the CLI to authenticate",
	}

	var showSensitive bool
	cmd.Flags().BoolVar(&showSensitive, "sensitive", false, "Include sensitive fields like passwords and tokens in the output")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := resolveProfileFromHostFlag(cmd, profile.DefaultProfiler); err != nil {
			return err
		}
		var status *authStatus
		var err error
		status, err = getAuthStatus(cmd, args, showSensitive, func(cmd *cobra.Command, args []string) (*config.Config, bool, error) {
			isAccount, err := root.MustAnyClient(cmd, args)
			return cmdctx.ConfigUsed(cmd.Context()), isAccount, err
		})
		if err != nil {
			return err
		}

		if status.Error != nil {
			return render(ctx, cmd, status, errorTemplate)
		}

		return render(ctx, cmd, status, authTemplate)
	}

	return cmd
}

// resolveProfileFromHostFlag translates an explicit --host into a --profile
// for `auth describe`. Without this, the downstream profile resolver ignores
// --host and either falls back to [__settings__].default_profile (silently
// describing a different host than the one the user named) or errors with the
// SDK's default-credentials message even though a host-matching profile
// exists. DATABRICKS_CONFIG_PROFILE is left alone — it's an explicit signal
// the user already made.
func resolveProfileFromHostFlag(cmd *cobra.Command, profiler profile.Profiler) error {
	hostFlag := cmd.Flag("host")
	profileFlag := cmd.Flag("profile")
	if hostFlag == nil || profileFlag == nil {
		return nil
	}
	if !hostFlag.Changed || profileFlag.Changed {
		return nil
	}
	ctx := cmd.Context()
	if env.Get(ctx, "DATABRICKS_CONFIG_PROFILE") != "" {
		return nil
	}
	profileName, err := resolveHostToProfile(ctx, hostFlag.Value.String(), profiler)
	if err != nil {
		return err
	}
	return profileFlag.Value.Set(profileName)
}

type tryAuth func(cmd *cobra.Command, args []string) (*config.Config, bool, error)

func getAuthStatus(cmd *cobra.Command, args []string, showSensitive bool, fn tryAuth) (*authStatus, error) {
	cfg, isAccount, err := fn(cmd, args)
	ctx := cmd.Context()
	if err != nil {
		details := getAuthDetails(cmd, cfg, showSensitive)
		return &authStatus{ //nolint:nilerr // error is returned in the authStatus struct
			Status:       "error",
			Error:        err,
			Details:      details,
			TokenStorage: resolveTokenStorageInfo(ctx, details.AuthType),
		}, nil
	}

	if isAccount {
		a := cmdctx.AccountClient(ctx)

		// Doing a simple API call to check if the auth is valid
		_, err := a.Workspaces.List(ctx)
		if err != nil {
			details := getAuthDetails(cmd, cfg, showSensitive)
			return &authStatus{ //nolint:nilerr // error is returned in the authStatus struct
				Status:       "error",
				Error:        err,
				Details:      details,
				TokenStorage: resolveTokenStorageInfo(ctx, details.AuthType),
			}, nil
		}

		details := getAuthDetails(cmd, a.Config, showSensitive)
		status := authStatus{
			Status:       "success",
			Details:      details,
			AccountID:    a.Config.AccountID,
			Username:     a.Config.Username,
			TokenStorage: resolveTokenStorageInfo(ctx, details.AuthType),
		}

		return &status, nil
	}

	w := cmdctx.WorkspaceClient(ctx)
	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		details := getAuthDetails(cmd, cfg, showSensitive)
		return &authStatus{ //nolint:nilerr // error is returned in the authStatus struct
			Status:       "error",
			Error:        err,
			Details:      details,
			TokenStorage: resolveTokenStorageInfo(ctx, details.AuthType),
		}, nil
	}

	details := getAuthDetails(cmd, w.Config, showSensitive)
	status := authStatus{
		Status:       "success",
		Details:      details,
		Username:     me.UserName,
		TokenStorage: resolveTokenStorageInfo(ctx, details.AuthType),
	}

	return &status, nil
}

func render(ctx context.Context, cmd *cobra.Command, status *authStatus, template string) error {
	switch root.OutputType(cmd) {
	case flags.OutputText:
		return cmdio.RenderWithTemplate(ctx, map[string]any{
			"Status":           status,
			"ConfigAttributes": config.ConfigAttributes,
		}, "", template)
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(status, "", "  ")
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(buf)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}

	return nil
}

type authStatus struct {
	Status       string             `json:"status"`
	Error        error              `json:"error,omitempty"`
	Username     string             `json:"username,omitempty"`
	AccountID    string             `json:"account_id,omitempty"`
	Details      config.AuthDetails `json:"details"`
	TokenStorage *tokenStorageInfo  `json:"token_storage,omitempty"`
}

// tokenStorageInfo describes where the U2M (databricks-cli) token cache lives
// and which precedence level produced that choice. Populated only when the
// active auth type is "databricks-cli"; other auth types do not use the
// CLI's U2M token cache and the field is omitted.
type tokenStorageInfo struct {
	Mode     string `json:"mode"`
	Location string `json:"location"`
	Source   string `json:"source"`
}

const (
	plaintextLocation = "~/.databricks/token-cache.json"
	secureLocation    = "OS keyring (service: databricks-cli)"
)

// resolveTokenStorageInfo returns storage info for the U2M token cache, or
// nil if the active auth type does not use the cache. Resolver errors are
// logged at debug level and treated as "no info available" rather than
// failing describe; the user-visible auth status is more important than
// secondary metadata about where a token would have been stored.
func resolveTokenStorageInfo(ctx context.Context, authType string) *tokenStorageInfo {
	if authType != authTypeDatabricksCLI {
		return nil
	}
	mode, source, err := storage.ResolveStorageModeWithSource(ctx, "")
	if err != nil {
		log.Debugf(ctx, "auth describe: resolve storage mode: %v", err)
		return nil
	}
	info := &tokenStorageInfo{
		Mode:   string(mode),
		Source: storageSourceLabel(ctx, source),
	}
	switch mode {
	case storage.StorageModePlaintext:
		info.Location = plaintextLocation
	case storage.StorageModeSecure:
		info.Location = secureLocation
	default:
		return nil
	}
	return info
}

// storageSourceLabel returns a user-facing label for source. For
// StorageSourceConfig, it appends the resolved config file path
// (DATABRICKS_CONFIG_FILE or <home>/.databrickscfg) so the output matches
// the SDK's config.Source style ("from <path> config file") rather than
// hardcoding ".databrickscfg" when a custom path is in use.
func storageSourceLabel(ctx context.Context, source storage.StorageSource) string {
	if source != storage.StorageSourceConfig {
		return source.String()
	}
	return "auth_storage in [__settings__] section of " + resolvedConfigPath(ctx)
}

// resolvedConfigPath returns the path the storage-mode resolver loaded from
// for [__settings__].auth_storage: DATABRICKS_CONFIG_FILE if set, otherwise
// <home>/.databrickscfg. Falls back to "~/.databrickscfg" only when the home
// directory cannot be determined (rare; describe should not crash on this
// secondary metadata path).
func resolvedConfigPath(ctx context.Context) string {
	if path := env.Get(ctx, "DATABRICKS_CONFIG_FILE"); path != "" {
		return path
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		log.Debugf(ctx, "auth describe: resolve home dir: %v", err)
		return "~/.databrickscfg"
	}
	return filepath.ToSlash(filepath.Join(home, ".databrickscfg"))
}

func getAuthDetails(cmd *cobra.Command, cfg *config.Config, showSensitive bool) config.AuthDetails {
	var opts []config.AuthDetailsOptions
	if showSensitive {
		opts = append(opts, config.ShowSensitive)
	}
	details := cfg.GetAuthDetails(opts...)

	for k, v := range details.Configuration {
		if k == "profile" && cmd.Flag("profile").Changed {
			v.Source = config.Source{Type: config.SourceType("flag"), Name: "--profile"}
		}

		if k == "host" && cmd.Flag("host").Changed {
			v.Source = config.Source{Type: config.SourceType("flag"), Name: "--host"}
		}
	}

	// If profile is not set explicitly, default to "default"
	if _, ok := details.Configuration["profile"]; !ok {
		profile := cfg.Profile
		if profile == "" {
			profile = "default"
		}
		details.Configuration["profile"] = &config.AttrConfig{Value: profile, Source: config.Source{Type: config.SourceDynamicConfig}}
	}

	// Unset source for databricks_cli_path because it can't be overridden anyway
	if v, ok := details.Configuration["databricks_cli_path"]; ok {
		v.Source = config.Source{Type: config.SourceDynamicConfig}
	}

	return details
}
