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
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	networkTimeout = 10 * time.Second
	checkTimeout   = 15 * time.Second
)

func pass(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: statusPass, Message: msg}
}

func info(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: statusInfo, Message: msg}
}

func warn(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: statusWarn, Message: msg}
}

func skip(name, msg string) CheckResult {
	return CheckResult{Name: name, Status: statusSkip, Message: msg}
}

func fail(name, msg string, err error) CheckResult {
	r := CheckResult{Name: name, Status: statusFail, Message: msg}
	if err != nil {
		r.Detail = err.Error()
	}
	return r
}

func withCheckTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, checkTimeout)
}

// runChecks runs all diagnostic checks and returns the results.
func runChecks(ctx context.Context, profile string, profileFromFlag bool) []CheckResult {
	cfg, err := resolveConfig(ctx, profile, profileFromFlag)
	authResult, authCfg := checkAuth(ctx, cfg, err)

	identityResult := skip("Identity", "Skipped (authentication failed)")
	if authCfg != nil {
		identityResult = checkIdentity(ctx, authCfg)
	}

	return []CheckResult{
		checkCLIVersion(),
		checkUpdates(ctx, http.DefaultClient, updateCheckURL),
		checkToolchain(ctx, realExec),
		checkProxy(ctx),
		checkLogFile(ctx),
		checkConfigFile(ctx),
		checkCurrentProfile(ctx, profile, profileFromFlag, cfg),
		authResult,
		identityResult,
		checkNetwork(ctx, cfg, err, authCfg),
	}
}

func checkCLIVersion() CheckResult {
	return info("CLI Version", build.GetInfo().Version)
}

func checkConfigFile(ctx context.Context) CheckResult {
	profiler := profile.GetProfiler(ctx)

	path, err := profiler.GetPath(ctx)
	if err != nil {
		return fail("Config File", "Cannot determine config file path", err)
	}

	profiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil {
		// Config file absence is not a hard failure since auth can work via env vars.
		if errors.Is(err, profile.ErrNoConfiguration) {
			return warn("Config File", "No config file found (auth can still work via environment variables)")
		}
		return fail("Config File", "Cannot read "+filepath.ToSlash(path), err)
	}

	return pass("Config File", fmt.Sprintf("%s (%d profiles)", filepath.ToSlash(path), len(profiles)))
}

func checkCurrentProfile(ctx context.Context, profileName string, fromFlag bool, resolvedCfg *config.Config) CheckResult {
	switch envProfile := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); {
	case fromFlag:
		return info("Current Profile", profileName)
	case envProfile != "":
		return info("Current Profile", envProfile+" (from DATABRICKS_CONFIG_PROFILE)")
	// The SDK resolves a profile when DEFAULT is in .databrickscfg.
	case resolvedCfg != nil && resolvedCfg.Profile != "":
		return info("Current Profile", resolvedCfg.Profile+" (resolved from config file)")
	default:
		return info("Current Profile", "none (using environment or defaults)")
	}
}

func resolveConfig(ctx context.Context, profileName string, fromFlag bool) (*config.Config, error) {
	cfg := &config.Config{
		Loaders: []config.Loader{
			env.NewConfigLoader(ctx),
			config.ConfigAttributes,
			config.ConfigFile,
		},
	}
	if fromFlag {
		cfg.Profile = profileName
	}
	return cfg, cfg.EnsureResolved()
}

// isAccountLevelConfig returns true if the resolved config targets account-level APIs.
// It uses auth.ResolveConfigType so SPOG / unified-host profiles, which the SDK's own
// ConfigType() classifies as InvalidConfig, are still recognised as account-level.
func isAccountLevelConfig(cfg *config.Config) bool {
	return auth.ResolveConfigType(cfg) == config.AccountConfig
}

// checkAuth uses the resolved config to authenticate.
// On success it returns the authenticated config for use in subsequent checks.
func checkAuth(ctx context.Context, cfg *config.Config, resolveErr error) (CheckResult, *config.Config) {
	ctx, cancel := withCheckTimeout(ctx)
	defer cancel()

	if resolveErr != nil {
		return fail("Authentication", "Cannot resolve config", resolveErr), nil
	}

	// Detect account-level configs and use the appropriate client constructor
	// so that account profiles are not incorrectly reported as broken.
	var authCfg *config.Config
	if isAccountLevelConfig(cfg) {
		a, err := databricks.NewAccountClient((*databricks.Config)(cfg))
		if err != nil {
			return fail("Authentication", "Cannot create account client", err), nil
		}
		authCfg = a.Config
	} else {
		w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
		if err != nil {
			return fail("Authentication", "Cannot create workspace client", err), nil
		}
		authCfg = w.Config
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, authCfg.Host, nil)
	if err != nil {
		return fail("Authentication", "Internal error", err), nil
	}

	if err := authCfg.Authenticate(req); err != nil {
		return fail("Authentication", "Authentication failed", err), nil
	}

	msg := fmt.Sprintf("OK (%s)", authCfg.AuthType)
	if isAccountLevelConfig(cfg) {
		msg += " [account-level]"
	}
	return pass("Authentication", msg), authCfg
}

func checkIdentity(ctx context.Context, authCfg *config.Config) CheckResult {
	ctx, cancel := withCheckTimeout(ctx)
	defer cancel()

	if isAccountLevelConfig(authCfg) {
		return checkAccountIdentity(ctx, authCfg)
	}

	w, err := databricks.NewWorkspaceClient((*databricks.Config)(authCfg))
	if err != nil {
		return fail("Identity", "Cannot create workspace client", err)
	}

	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return fail("Identity", "Cannot fetch current user", err)
	}

	return pass("Identity", me.UserName)
}

// checkAccountIdentity issues a lightweight authenticated account API call so
// account-level profiles get server-side credential validation instead of being
// skipped (which would let invalid account PAT/OAuth report Authentication: OK).
func checkAccountIdentity(ctx context.Context, authCfg *config.Config) CheckResult {
	a, err := databricks.NewAccountClient((*databricks.Config)(authCfg))
	if err != nil {
		return fail("Identity", "Cannot create account client", err)
	}

	workspaces, err := a.Workspaces.List(ctx)
	if err != nil {
		return fail("Identity", "Cannot list account workspaces", err)
	}

	return pass("Identity", fmt.Sprintf("account %s (%d workspaces)", authCfg.AccountID, len(workspaces)))
}

func checkNetwork(ctx context.Context, cfg *config.Config, resolveErr error, authCfg *config.Config) CheckResult {
	// Prefer the authenticated config (it has the fully resolved host).
	if authCfg != nil {
		return checkNetworkWithHost(ctx, authCfg.Host, configuredNetworkHTTPClient(authCfg))
	}

	// Auth failed or was skipped. If we still have a host from config resolution
	// (even if resolution had other errors), attempt the network check.
	if cfg != nil && cfg.Host != "" {
		log.Warnf(ctx, "authenticated client unavailable for network check, using config-based HTTP client")
		return checkNetworkWithHost(ctx, cfg.Host, configuredNetworkHTTPClient(cfg))
	}

	return fail("Network", "No host configured", resolveErr)
}

func checkNetworkWithHost(ctx context.Context, host string, client *http.Client) CheckResult {
	ctx, cancel := context.WithTimeout(ctx, networkTimeout)
	defer cancel()

	if host == "" {
		return fail("Network", "No host configured", nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, host, nil)
	if err != nil {
		return fail("Network", "Cannot create request for "+host, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fail("Network", "Cannot reach "+host, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	return pass("Network", host+" is reachable")
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
