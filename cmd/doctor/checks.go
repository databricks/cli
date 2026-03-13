package doctor

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	statusPass = "pass"
	statusFail = "fail"
	statusWarn = "warn"
	statusInfo = "info"
	statusSkip = "skip"

	networkTimeout = 10 * time.Second
	checkTimeout   = 15 * time.Second
)

// runChecks runs all diagnostic checks and returns the results.
func runChecks(cmd *cobra.Command) []CheckResult {
	cfg, err := resolveConfig(cmd)

	var results []CheckResult

	results = append(results, checkCLIVersion())
	results = append(results, checkConfigFile(cmd))
	results = append(results, checkCurrentProfile(cmd))

	authResult, authCfg := checkAuth(cmd, cfg, err)
	results = append(results, authResult)

	if authCfg != nil {
		results = append(results, checkIdentity(cmd, authCfg))
	} else {
		results = append(results, CheckResult{
			Name:    "Identity",
			Status:  statusSkip,
			Message: "Skipped (authentication failed)",
		})
	}

	results = append(results, checkNetwork(cmd, cfg, err, authCfg))
	return results
}

func checkCLIVersion() CheckResult {
	info := build.GetInfo()
	return CheckResult{
		Name:    "CLI Version",
		Status:  statusInfo,
		Message: info.Version,
	}
}

func checkConfigFile(cmd *cobra.Command) CheckResult {
	ctx := cmd.Context()
	profiler := profile.GetProfiler(ctx)

	path, err := profiler.GetPath(ctx)
	if err != nil {
		return CheckResult{
			Name:    "Config File",
			Status:  statusFail,
			Message: "Cannot determine config file path",
			Detail:  err.Error(),
		}
	}

	profiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil {
		// Config file absence is not a hard failure since auth can work via env vars.
		if errors.Is(err, profile.ErrNoConfiguration) {
			return CheckResult{
				Name:    "Config File",
				Status:  statusWarn,
				Message: "No config file found (auth can still work via environment variables)",
			}
		}
		return CheckResult{
			Name:    "Config File",
			Status:  statusFail,
			Message: "Cannot read " + path,
			Detail:  err.Error(),
		}
	}

	return CheckResult{
		Name:    "Config File",
		Status:  statusPass,
		Message: fmt.Sprintf("%s (%d profiles)", path, len(profiles)),
	}
}

func checkCurrentProfile(cmd *cobra.Command) CheckResult {
	ctx := cmd.Context()

	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && profileFlag.Changed {
		return CheckResult{
			Name:    "Current Profile",
			Status:  statusInfo,
			Message: profileFlag.Value.String(),
		}
	}

	if envProfile := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); envProfile != "" {
		return CheckResult{
			Name:    "Current Profile",
			Status:  statusInfo,
			Message: envProfile + " (from DATABRICKS_CONFIG_PROFILE)",
		}
	}

	return CheckResult{
		Name:    "Current Profile",
		Status:  statusInfo,
		Message: "none (using environment or defaults)",
	}
}

func resolveConfig(cmd *cobra.Command) (*config.Config, error) {
	ctx := cmd.Context()
	cfg := &config.Config{}

	if configFile := env.Get(ctx, "DATABRICKS_CONFIG_FILE"); configFile != "" {
		cfg.ConfigFile = configFile
	} else if home := env.Get(ctx, env.HomeEnvVar()); home != "" {
		cfg.ConfigFile = filepath.Join(home, ".databrickscfg")
	}

	cfg.Loaders = []config.Loader{
		env.NewConfigLoader(ctx),
		config.ConfigAttributes,
		config.ConfigFile,
	}

	profileFlag := cmd.Flag("profile")
	if profileFlag != nil && profileFlag.Changed {
		cfg.Profile = profileFlag.Value.String()
	}

	return cfg, cfg.EnsureResolved()
}

// isAccountLevelConfig returns true if the resolved config targets account-level APIs.
func isAccountLevelConfig(cfg *config.Config) bool {
	return cfg.AccountID != "" && cfg.Host != "" && cfg.HostType() == config.AccountHost
}

// checkAuth uses the resolved config to authenticate.
// On success it returns the authenticated config for use in subsequent checks.
func checkAuth(cmd *cobra.Command, cfg *config.Config, resolveErr error) (CheckResult, *config.Config) {
	ctx, cancel := context.WithTimeout(cmd.Context(), checkTimeout)
	defer cancel()

	if resolveErr != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  statusFail,
			Message: "Cannot resolve config",
			Detail:  resolveErr.Error(),
		}, nil
	}

	// Detect account-level configs and use the appropriate client constructor
	// so that account profiles are not incorrectly reported as broken.
	var authCfg *config.Config
	if isAccountLevelConfig(cfg) {
		a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
		if err != nil {
			return CheckResult{
				Name:    "Authentication",
				Status:  statusFail,
				Message: "Cannot create account client",
				Detail:  err.Error(),
			}, nil
		}
		authCfg = a.Config
	} else {
		w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err != nil {
			return CheckResult{
				Name:    "Authentication",
				Status:  statusFail,
				Message: "Cannot create workspace client",
				Detail:  err.Error(),
			}, nil
		}
		authCfg = w.Config
	}

	req, err := http.NewRequestWithContext(ctx, "", "", nil)
	if err != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  statusFail,
			Message: "Internal error",
			Detail:  err.Error(),
		}, nil
	}

	err = authCfg.Authenticate(req)
	if err != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  statusFail,
			Message: "Authentication failed",
			Detail:  err.Error(),
		}, nil
	}

	msg := fmt.Sprintf("OK (%s)", authCfg.AuthType)
	if isAccountLevelConfig(cfg) {
		msg += " [account-level]"
	}

	return CheckResult{
		Name:    "Authentication",
		Status:  statusPass,
		Message: msg,
	}, authCfg
}

func checkIdentity(cmd *cobra.Command, authCfg *config.Config) CheckResult {
	ctx, cancel := context.WithTimeout(cmd.Context(), checkTimeout)
	defer cancel()

	// Account-level configs don't support the /me endpoint for workspace identity.
	if authCfg.HostType() == config.AccountHost {
		return CheckResult{
			Name:    "Identity",
			Status:  statusSkip,
			Message: "Skipped (account-level profile, workspace identity not available)",
		}
	}

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(authCfg))
	if err != nil {
		return CheckResult{
			Name:    "Identity",
			Status:  statusFail,
			Message: "Cannot create workspace client",
			Detail:  err.Error(),
		}
	}

	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return CheckResult{
			Name:    "Identity",
			Status:  statusFail,
			Message: "Cannot fetch current user",
			Detail:  err.Error(),
		}
	}

	return CheckResult{
		Name:    "Identity",
		Status:  statusPass,
		Message: me.UserName,
	}
}

func checkNetwork(cmd *cobra.Command, cfg *config.Config, resolveErr error, authCfg *config.Config) CheckResult {
	// Prefer the authenticated config (it has the fully resolved host).
	if authCfg != nil {
		return checkNetworkWithHost(cmd, authCfg.Host, configuredNetworkHTTPClient(authCfg))
	}

	// Auth failed or was skipped. If we still have a host from config resolution
	// (even if resolution had other errors), attempt the network check.
	if cfg != nil && cfg.Host != "" {
		log.Warnf(cmd.Context(), "authenticated client unavailable for network check, using config-based HTTP client")
		return checkNetworkWithHost(cmd, cfg.Host, configuredNetworkHTTPClient(cfg))
	}

	// No host available at all.
	detail := "no host configured"
	if resolveErr != nil {
		detail = resolveErr.Error()
	}
	return CheckResult{
		Name:    "Network",
		Status:  statusFail,
		Message: "No host configured",
		Detail:  detail,
	}
}

func checkNetworkWithHost(cmd *cobra.Command, host string, client *http.Client) CheckResult {
	ctx, cancel := context.WithTimeout(cmd.Context(), networkTimeout)
	defer cancel()

	if host == "" {
		return CheckResult{
			Name:    "Network",
			Status:  statusFail,
			Message: "No host configured",
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, host, nil)
	if err != nil {
		return CheckResult{
			Name:    "Network",
			Status:  statusFail,
			Message: "Cannot create request for " + host,
			Detail:  err.Error(),
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return CheckResult{
			Name:    "Network",
			Status:  statusFail,
			Message: "Cannot reach " + host,
			Detail:  err.Error(),
		}
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	return CheckResult{
		Name:    "Network",
		Status:  statusPass,
		Message: host + " is reachable",
	}
}

func configuredNetworkHTTPClient(cfg *config.Config) *http.Client {
	return &http.Client{
		Transport: configuredNetworkHTTPTransport(cfg),
	}
}

func configuredNetworkHTTPTransport(cfg *config.Config) http.RoundTripper {
	if cfg.HTTPTransport != nil {
		return cfg.HTTPTransport
	}

	if !cfg.InsecureSkipVerify {
		return http.DefaultTransport
	}

	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return http.DefaultTransport
	}

	clone := transport.Clone()
	if clone.TLSClientConfig != nil {
		clone.TLSClientConfig = clone.TLSClientConfig.Clone()
	} else {
		clone.TLSClientConfig = &tls.Config{}
	}
	clone.TLSClientConfig.InsecureSkipVerify = true
	return clone
}
