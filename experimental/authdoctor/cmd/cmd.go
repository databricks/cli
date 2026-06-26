// Package cmd wires the auth doctor engine into the experimental command tree.
// It owns all the I/O (config resolution, token cache, keyring, env, network
// probe) and feeds a plain Snapshot to authdoctor.Analyze, which produces the
// verdict deterministically.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/authdoctor"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// probeTimeout bounds the best-effort connectivity probe so the doctor never
// blocks for long on an unreachable host.
const probeTimeout = 30 * time.Second

// SDK auth-type strings used for local auth-type inference.
const (
	authTypePAT = "pat"
	authTypeM2M = "oauth-m2m"
	// authTypeUnknown marks a profile that carries non-U2M credentials we don't
	// classify further; it gates out the U2M-only checks without claiming a type.
	authTypeUnknown = "unknown"
	patPrefix       = "dapi"
	patTokenName    = "DATABRICKS_TOKEN"
)

// probeAuth performs the connectivity probe. It is a package variable so tests
// can stub out the network call.
var probeAuth = runProbe

// New returns the experimental "auth" command group with the doctor subcommand.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication diagnostics",
		Long:  "Authentication diagnostics that explain why auth is broken and how to fix it.",
	}
	cmd.AddCommand(newDoctorCommand())
	return cmd
}

func newDoctorCommand() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose why authentication is broken and print the fix",
		Long: `Diagnose authentication for the active (or selected) profile and print ranked
findings, each with a one-line fix.

The doctor is best-effort and never aborts on a failed login: it resolves the
configuration locally, reads the cached OAuth token, probes the keyring, and
reports. In single-profile mode it also makes a best-effort authenticated call
(bounded by a short timeout) to validate the credentials end-to-end and surface
OAuth scope errors; --all skips that network probe.

Use --profile to target a specific profile or --all for every profile.`,
		Args: cobra.NoArgs,
		// No PreRunE: the doctor must run even when auth is broken, so it does
		// not gate on root.MustWorkspaceClient.
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			var reports []authdoctor.Report
			if all {
				profiles, err := profile.GetProfiler(ctx).LoadProfiles(ctx, profile.MatchAllProfiles)
				if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
					// Honor the never-abort contract: report the read failure as a
					// finding rather than returning a bare error.
					reports = append(reports, configErrorReport(err))
				}
				for _, p := range profiles {
					s := buildSnapshot(ctx, cmd, p.Name, sourceAll, false)
					reports = append(reports, authdoctor.Analyze(s))
				}
			} else {
				name, src := resolveProfile(ctx, cmd)
				s := buildSnapshot(ctx, cmd, name, src, true)
				reports = append(reports, authdoctor.Analyze(s))
			}

			return renderReports(ctx, cmd, reports)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Diagnose every profile in the config file")
	return cmd
}

// sourceAll is the ProfileSource label for profiles selected by --all.
const sourceAll = "--all"

// configErrorReport wraps a config-file read failure as a report so --all keeps
// the doctor's never-abort contract instead of returning a bare error.
func configErrorReport(err error) authdoctor.Report {
	return authdoctor.Report{
		Overall: authdoctor.LevelError,
		Findings: []authdoctor.Finding{{
			Check:  "resolution",
			Level:  authdoctor.LevelError,
			Title:  "Could not read the config file",
			Detail: err.Error(),
			Fix:    "check ~/.databrickscfg (or DATABRICKS_CONFIG_FILE) for syntax errors",
		}},
	}
}

// resolveProfile mirrors the normal-command profile precedence (Path A) and
// reports which source produced the profile, so the doctor can flag the
// [DEFAULT]-section case that `databricks api` resolves differently.
func resolveProfile(ctx context.Context, cmd *cobra.Command) (name, source string) {
	if f := cmd.Flag("profile"); f != nil && f.Changed && f.Value.String() != "" {
		return f.Value.String(), authdoctor.SourceFlag
	}
	if p := env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); p != "" {
		return p, authdoctor.SourceEnvProfile
	}
	// Pass the DATABRICKS_CONFIG_FILE path explicitly: GetConfiguredDefaultProfile
	// defaults an empty path to ~/.databrickscfg, while ResolveDefaultProfile and
	// the SDK honor the env var. Reading different files would let the doctor
	// report a profile the SDK never loads.
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	if dp, err := databrickscfg.GetConfiguredDefaultProfile(ctx, configPath); err == nil && dp != "" {
		return dp, authdoctor.SourceDefaultProfile
	}
	if resolved := databrickscfg.ResolveDefaultProfile(ctx); resolved != "" {
		return resolved, authdoctor.SourceDefaultSection
	}
	return "", authdoctor.SourceNone
}

// buildSnapshot assembles the locally observable auth state for one profile.
// All I/O happens here; the analysis in authdoctor.Analyze stays pure.
func buildSnapshot(ctx context.Context, cmd *cobra.Command, profileName, source string, probe bool) authdoctor.Snapshot {
	s := authdoctor.Snapshot{
		Profile:       profileName,
		ProfileSource: source,
		Now:           time.Now(),
		EnvHost:       env.Get(ctx, "DATABRICKS_HOST"),
		EnvToken:      env.Get(ctx, patTokenName),
	}

	// Resolve config locally. EnsureResolved would otherwise fetch host
	// metadata over the network (and can block for minutes on an unreachable
	// host); a no-op resolver keeps the doctor fast and offline-safe.
	cfg := &config.Config{
		Profile:              profileName,
		HostMetadataResolver: func(context.Context, string) (*config.HostMetadata, error) { return nil, nil },
	}
	if err := cfg.EnsureResolved(); err != nil {
		s.ConfigError = err.Error()
	}
	s.Host = cfg.Host
	s.AccountID = cfg.AccountID
	s.HostType = classifyHostType(cfg.Host)
	s.AuthType = effectiveAuthType(cfg)
	// cfg.AuthType is set only when explicitly pinned (profile auth_type or
	// DATABRICKS_AUTH_TYPE); inference does not populate it.
	s.ExplicitAuthType = cfg.AuthType
	s.PATPresent = cfg.Token != ""
	s.PATLooksValid = strings.HasPrefix(cfg.Token, patPrefix)
	s.ClientIDPresent = cfg.ClientID != ""
	s.ClientSecretPresent = cfg.ClientSecret != ""

	// The profile's declared host (pre env-override) and persisted OAuth scopes
	// come from the config file, not the resolved config.
	if profileName != "" {
		if p, ok := loadProfileEntry(ctx, profileName); ok {
			s.ProfileHost = p.Host
			s.ConfiguredScopes = splitScopes(p.Scopes)
		}
	}

	s.BundlePath, s.BundleHost, s.BundleProfile = detectBundle(ctx)

	if authdoctor.IsU2M(s.AuthType) {
		s.Token = lookupToken(ctx, profileName)
		fillStorageState(ctx, &s)
	}

	if probe && s.Host != "" {
		s.ProbeAttempted = true
		res := probeAuth(ctx, profileName, s.HostType, s.Host)
		s.ProbeUser = res.user
		s.ProbeErr = res.err
		if res.hostType != "" {
			s.HostType = res.hostType
		}
		if res.clockSkewKnown {
			s.ClockSkew = res.clockSkew
			s.ClockSkewKnown = true
		}
		// Reuse the CLI's shared remediation copy for ordinary auth failures so
		// the doctor's advice matches what every other command prints. Scope
		// errors keep the doctor's downscoping-specific fix instead.
		if s.ProbeErr != nil {
			if _, _, ok := authdoctor.ClassifyScopeError(s.ProbeErr); !ok {
				s.ProbeErr = auth.EnrichAuthError(ctx, cfg, s.ProbeErr)
			}
		}
	}

	return s
}

// bundleFileNames are the bundle root marker filenames, matching
// bundle/config.FileNames.
var bundleFileNames = []string{"databricks.yml", "databricks.yaml"}

// detectBundle finds the nearest databricks.yml in the working-directory tree
// (honoring DATABRICKS_BUNDLE_ROOT) and reads its top-level workspace.host and
// workspace.profile, best-effort. Targets, includes, and variables are not
// expanded; values are returned verbatim so the engine can flag unevaluated
// "${...}" references rather than compare them. Returns empty strings when no
// bundle is present.
func detectBundle(ctx context.Context) (path, host, profileName string) {
	root := bundleRoot(ctx)
	if root == "" {
		return "", "", ""
	}
	for _, name := range bundleFileNames {
		candidate := filepath.Join(root, name)
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		host, profileName = readBundleWorkspace(ctx, candidate)
		return filepath.ToSlash(candidate), host, profileName
	}
	return "", "", ""
}

// bundleRoot returns the directory holding the bundle: DATABRICKS_BUNDLE_ROOT
// when set, otherwise the nearest ancestor of the working directory containing
// a bundle file.
func bundleRoot(ctx context.Context) string {
	if r := env.Get(ctx, "DATABRICKS_BUNDLE_ROOT"); r != "" {
		return r
	}
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		for _, name := range bundleFileNames {
			if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// readBundleWorkspace reads workspace.host and workspace.profile from a bundle
// file. Read failures are non-fatal (the doctor still reports the bundle's
// presence), logged at debug.
func readBundleWorkspace(ctx context.Context, path string) (host, profileName string) {
	contents, err := os.ReadFile(path)
	if err != nil {
		log.Debugf(ctx, "auth doctor: read bundle %s: %v", path, err)
		return "", ""
	}
	v, err := yamlloader.LoadYAML(path, strings.NewReader(string(contents)))
	if err != nil {
		log.Debugf(ctx, "auth doctor: parse bundle %s: %v", path, err)
		return "", ""
	}
	return bundleString(v, "host"), bundleString(v, "profile")
}

// bundleString reads workspace.<key> as a string from a bundle dyn.Value.
func bundleString(v dyn.Value, key string) string {
	got, err := dyn.Get(v, "workspace."+key)
	if err != nil {
		return ""
	}
	s, ok := got.AsString()
	if !ok {
		return ""
	}
	return s
}

// loadProfileEntry loads a single profile's config-file entry by name.
func loadProfileEntry(ctx context.Context, name string) (profile.Profile, bool) {
	profiles, err := profile.GetProfiler(ctx).LoadProfiles(ctx, profile.WithName(name))
	if err != nil || len(profiles) == 0 {
		return profile.Profile{}, false
	}
	return profiles[0], true
}

// lookupToken reads the cached U2M token for a profile. The cache key is the
// profile name, so a host-only configuration (no profile) cannot be keyed and
// returns an empty state.
func lookupToken(ctx context.Context, profileName string) authdoctor.TokenState {
	if profileName == "" {
		return authdoctor.TokenState{}
	}
	store, mode, err := storage.ResolveStore(ctx, "")
	if err != nil {
		return authdoctor.TokenState{LookupErr: err.Error()}
	}
	tc := storage.OAuthTokenCache(ctx, store, mode)
	tok, err := tc.Lookup(profileName)
	if err != nil {
		if errors.Is(err, cache.ErrNotFound) {
			return authdoctor.TokenState{NotFoundHint: storage.HintForNotFound(err)}
		}
		return authdoctor.TokenState{LookupErr: err.Error()}
	}
	return authdoctor.TokenState{
		Found:      true,
		Expiry:     tok.Expiry,
		HasRefresh: tok.RefreshToken != "",
	}
}

// fillStorageState records the token-cache backend and, for secure storage,
// whether the OS keyring is reachable.
func fillStorageState(ctx context.Context, s *authdoctor.Snapshot) {
	mode, src, err := storage.ResolveStorageModeWithSource(ctx, "")
	if err != nil {
		log.Debugf(ctx, "auth doctor: resolve storage mode: %v", err)
		return
	}
	s.StorageMode = string(mode)
	s.StorageSource = src.String()
	if mode == storage.StorageModeSecure {
		probeErr := storage.ProbeKeyringRead()
		reachable := probeErr == nil
		s.KeyringReachable = &reachable
		if probeErr != nil {
			s.KeyringProbeErr = probeErr.Error()
		}
	}
}

// probeResult is the outcome of the best-effort connectivity probe.
type probeResult struct {
	user string
	// hostType is the metadata-refined classification (may be "unified"); empty
	// keeps the prefix-based classification.
	hostType       string
	clockSkew      time.Duration
	clockSkewKnown bool
	err            error
}

// runProbe makes a best-effort identity call to validate the credentials
// end-to-end. It builds a fresh client config from the profile so the SDK
// performs real OIDC discovery (needed to refresh an expired U2M token) rather
// than the no-op host-metadata resolver buildSnapshot uses for offline-safe
// local resolution.
//
// The whole probe runs in a goroutine bounded by probeTimeout: client
// construction triggers the SDK's host-metadata fetch, which uses its own
// background context and would otherwise be unbounded on an unreachable host.
// On timeout the goroutine is abandoned (acceptable in a short-lived CLI).
func runProbe(ctx context.Context, profileName, hostType, host string) probeResult {
	ch := make(chan probeResult, 1)
	go func() {
		ch <- doProbe(ctx, profileName, hostType, host)
	}()

	select {
	case r := <-ch:
		return r
	case <-time.After(probeTimeout):
		return probeResult{err: fmt.Errorf("authentication check timed out after %s", probeTimeout)}
	}
}

// doProbe builds a client for the profile, calls an identity endpoint, and
// gathers the metadata-refined host type and clock skew along the way.
func doProbe(ctx context.Context, profileName, hostType, host string) probeResult {
	ctx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()

	res := probeResult{}
	if skew, ok := measureClockSkew(ctx, host); ok {
		res.clockSkew = skew
		res.clockSkewKnown = true
	}

	if hostType == authdoctor.HostTypeAccount {
		a, err := databricks.NewAccountClient(&databricks.Config{Profile: profileName})
		if err != nil {
			res.err = err
			return res
		}
		// Account hosts have no Me endpoint; a successful list validates the
		// credentials, but there is no user identity to report.
		_, res.err = a.Workspaces.List(ctx)
		return res
	}

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Profile: profileName})
	if err != nil {
		res.err = err
		return res
	}
	me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		res.err = err
		return res
	}
	res.user = me.UserName
	// Refine the host classification from resolved metadata (detects unified
	// SPOG hosts that the URL-prefix heuristic cannot).
	res.hostType = classifyViaMetadata(ctx, w.Config)
	return res
}

// classifyViaMetadata maps the SDK host-metadata host type to the doctor's
// labels, or returns "" when it cannot be resolved.
func classifyViaMetadata(ctx context.Context, cfg *config.Config) string {
	meta, err := cfg.DefaultHostMetadataResolver()(ctx, cfg.CanonicalHostName())
	if err != nil || meta == nil {
		return ""
	}
	switch meta.HostType {
	case config.UnifiedHost:
		return authdoctor.HostTypeUnified
	case config.AccountHost:
		return authdoctor.HostTypeAccount
	case config.WorkspaceHost:
		return authdoctor.HostTypeWorkspace
	default:
		return ""
	}
}

// measureClockSkew reads the server's Date header via a HEAD request and returns
// local-minus-server time. Best-effort: any failure returns ok=false.
func measureClockSkew(ctx context.Context, host string) (time.Duration, bool) {
	if host == "" {
		return 0, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, host, nil)
	if err != nil {
		return 0, false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	serverTime, err := http.ParseTime(resp.Header.Get("Date"))
	if err != nil {
		return 0, false
	}
	return time.Since(serverTime), true
}

// effectiveAuthType returns the auth type the doctor should reason about. The
// SDK only assigns the definitive auth type while authenticating (which the
// doctor avoids to stay local), so when it is not set explicitly we infer it
// from the credentials that are present. We only default to databricks-cli
// (U2M) when there is no other credential signal at all; a profile carrying
// basic, Azure, Google, or metadata-service config authenticates through one of
// those strategies and must not be diagnosed as a U2M profile missing a token.
func effectiveAuthType(cfg *config.Config) string {
	if cfg.AuthType != "" {
		return cfg.AuthType
	}
	switch {
	case cfg.Token != "":
		return authTypePAT
	case cfg.ClientID != "" && cfg.ClientSecret != "":
		return authTypeM2M
	case hasNonU2MAuthSignal(cfg):
		return authTypeUnknown
	default:
		return authdoctor.AuthTypeDatabricksCLI
	}
}

// hasNonU2MAuthSignal reports whether the config carries credentials for a
// non-U2M strategy (basic, Azure, Google, or metadata service). Field set per
// the SDK config attributes (databricks-sdk-go/config/config.go).
func hasNonU2MAuthSignal(cfg *config.Config) bool {
	return cfg.Username != "" || cfg.Password != "" ||
		cfg.AzureResourceID != "" || cfg.AzureClientID != "" ||
		cfg.AzureClientSecret != "" || cfg.AzureTenantID != "" ||
		cfg.GoogleServiceAccount != "" || cfg.GoogleCredentials != "" ||
		cfg.MetadataServiceURL != ""
}

// accountHostPrefixes are the host prefixes that identify the account console.
// This mirrors the heuristic in the SDK's (now deprecated) Config.HostType; the
// doctor classifies the host for display, which is precisely the URL-pattern
// use the SDK deprecation steers product code away from.
var accountHostPrefixes = []string{"https://accounts.", "https://accounts-dod."}

// classifyHostType labels a host as the account console or a workspace by URL
// pattern. It does not detect unified hosts (that needs a network metadata
// lookup the doctor avoids).
func classifyHostType(host string) string {
	h := strings.ToLower(strings.TrimSpace(host))
	if h != "" && !strings.Contains(h, "://") {
		h = "https://" + h
	}
	for _, prefix := range accountHostPrefixes {
		if strings.HasPrefix(h, prefix) {
			return authdoctor.HostTypeAccount
		}
	}
	return authdoctor.HostTypeWorkspace
}

// splitScopes splits the comma-separated scopes persisted on a profile.
func splitScopes(scopes string) []string {
	var out []string
	for s := range strings.SplitSeq(scopes, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func renderReports(ctx context.Context, cmd *cobra.Command, reports []authdoctor.Report) error {
	switch root.OutputType(cmd) {
	case flags.OutputText:
		renderText(ctx, cmd, reports)
		return nil
	case flags.OutputJSON:
		buf, err := json.MarshalIndent(reports, "", "  ")
		if err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write(append(buf, '\n'))
		return err
	default:
		return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
	}
}

func renderText(ctx context.Context, cmd *cobra.Command, reports []authdoctor.Report) {
	out := cmd.OutOrStdout()
	if len(reports) == 0 {
		fmt.Fprintln(out, "No profiles found.")
		return
	}
	for i, r := range reports {
		if i > 0 {
			fmt.Fprintln(out)
		}
		header := fmt.Sprintf("Profile %q  host=%s  auth=%s", emptyDash(r.Profile), emptyDash(r.Host), emptyDash(r.AuthType))
		fmt.Fprintln(out, cmdio.Bold(ctx, header))
		for _, f := range r.Findings {
			fmt.Fprintf(out, "%s %s\n", glyph(ctx, f.Level), f.Title)
			if f.Detail != "" {
				fmt.Fprintf(out, "    %s\n", f.Detail)
			}
			if f.Fix != "" {
				fmt.Fprintf(out, "    Fix: %s\n", cmdio.Cyan(ctx, f.Fix))
			}
		}
		fmt.Fprintf(out, "Overall: %s\n", coloredLevel(ctx, r.Overall))
	}
}

// glyph returns the colored status glyph for a finding level.
func glyph(ctx context.Context, l authdoctor.Level) string {
	switch l {
	case authdoctor.LevelOK:
		return cmdio.Green(ctx, "✓")
	case authdoctor.LevelWarn:
		return cmdio.Yellow(ctx, "⚠")
	default:
		return cmdio.Red(ctx, "✗")
	}
}

func coloredLevel(ctx context.Context, l authdoctor.Level) string {
	switch l {
	case authdoctor.LevelOK:
		return cmdio.Green(ctx, "OK")
	case authdoctor.LevelWarn:
		return cmdio.Yellow(ctx, "WARNING")
	default:
		return cmdio.Red(ctx, "ERROR")
	}
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
