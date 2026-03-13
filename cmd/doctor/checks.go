package doctor

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

const (
	statusPass = "pass"
	statusFail = "fail"
	statusWarn = "warn"
	statusInfo = "info"

	networkTimeout = 10 * time.Second
)

// runChecks runs all diagnostic checks and returns the results.
func runChecks(cmd *cobra.Command) []CheckResult {
	cfg, err := resolveConfig(cmd)

	var results []CheckResult

	results = append(results, checkCLIVersion())
	results = append(results, checkConfigFile(cmd))
	results = append(results, checkCurrentProfile(cmd))

	authResult, w := checkAuth(cmd, cfg, err)
	results = append(results, authResult)

	if w != nil {
		results = append(results, checkIdentity(cmd, w))
	} else {
		results = append(results, CheckResult{
			Name:    "Identity",
			Status:  statusFail,
			Message: "Skipped (authentication failed)",
		})
	}

	results = append(results, checkNetwork(cmd, cfg, err, w))
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

// checkAuth uses the resolved config to authenticate.
func checkAuth(cmd *cobra.Command, cfg *config.Config, resolveErr error) (CheckResult, *databricks.WorkspaceClient) {
	ctx := cmd.Context()
	if resolveErr != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  statusFail,
			Message: "Cannot resolve config",
			Detail:  resolveErr.Error(),
		}, nil
	}

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  statusFail,
			Message: "Cannot create workspace client",
			Detail:  err.Error(),
		}, nil
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

	err = w.Config.Authenticate(req)
	if err != nil {
		return CheckResult{
			Name:    "Authentication",
			Status:  statusFail,
			Message: "Authentication failed",
			Detail:  err.Error(),
		}, nil
	}

	return CheckResult{
		Name:    "Authentication",
		Status:  statusPass,
		Message: fmt.Sprintf("OK (%s)", w.Config.AuthType),
	}, w
}

func checkIdentity(cmd *cobra.Command, w *databricks.WorkspaceClient) CheckResult {
	ctx := cmd.Context()
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

func checkNetwork(cmd *cobra.Command, cfg *config.Config, resolveErr error, w *databricks.WorkspaceClient) CheckResult {
	if resolveErr != nil {
		return CheckResult{
			Name:    "Network",
			Status:  statusFail,
			Message: "Cannot resolve config",
			Detail:  resolveErr.Error(),
		}
	}

	if w != nil {
		return checkNetworkWithHost(cmd, w.Config.Host)
	}

	return checkNetworkWithHost(cmd, cfg.Host)
}

func checkNetworkWithHost(cmd *cobra.Command, host string) CheckResult {
	ctx := cmd.Context()

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

	client := &http.Client{Timeout: networkTimeout}
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
	io.Copy(io.Discard, resp.Body)

	return CheckResult{
		Name:    "Network",
		Status:  statusPass,
		Message: host + " is reachable",
	}
}
