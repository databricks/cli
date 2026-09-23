package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

const (
	// home-relative directory to hold files/dependencies for the shim.
	agentDir = ".agent-shim"

	// directory which holds the per-agent wrappers; must match remoteShimDir in client.go.
	binDir = agentDir + "/bin"

	// the GitHub repo the shim installs the Unity Gateway CLI from.
	ugRepo = "databricks/unity-gateway"

	// pins the Unity Gateway CLI release tag the shim installs.
	ugVersion = "v0.1.0"

	// environment variable used to pass the user's workspace home to the shim for the agent context.
	workspaceHomeEnv = "DATABRICKS_WORKSPACE_HOME"

	// directory which holds toolchain deps (Node/npm) fetched when the image ships none.
	depsDir = agentDir + "/deps"

	// the file holding the Databricks session context.
	contextFile = "agent-system-context.md"

	// lock file to serialize first-run toolchain setup across
	// concurrent SSH clients sharing this driver's $HOME.
	setupLockName = ".setup.lock"

	// timeout to reclaim a setup lock left behind by a process that died
	// mid-install. It is deliberately generous: a cold first run can download uv,
	// ug, and Node
	setupLockStaleAfter = 2 * time.Minute
)

type agentSpec struct {
	name string
	// contextFlag passes the context file via this CLI flag (Claude's --append-system-prompt-file).
	contextFlag string
	// contextHomeFile writes the context to this $HOME instructions file (codex's AGENTS.md).
	contextHomeFile string
}

// Currently limited to the agents whose Unity Gateway CLI command supports --workspace
var supportedAgents = []agentSpec{
	{name: "claude", contextFlag: "--append-system-prompt-file"},
	{name: "codex", contextHomeFile: ".codex/AGENTS.md"},
}

// SupportedAgentNames lists the agents the ssh command registers a subcommand for.
func SupportedAgentNames() []string {
	names := make([]string, len(supportedAgents))
	for i, a := range supportedAgents {
		names[i] = a.name
	}
	return names
}

func agentByName(name string) (agentSpec, bool) {
	for _, a := range supportedAgents {
		if a.name == name {
			return a, true
		}
	}
	return agentSpec{}, false
}

func RunAgentShim(ctx context.Context, client *databricks.WorkspaceClient, agentName string, agentArgs []string) error {
	agent, ok := agentByName(agentName)
	if !ok {
		return fmt.Errorf("unsupported agent %q", agentName)
	}
	// Probe first: fail fast before the slow first-run bootstrap if the gateway is off.
	if err := probeAIGateway(ctx, client); err != nil {
		return err
	}
	workspace := strings.TrimRight(client.Config.Host, "/")
	return bootstrapAndLaunchAgent(ctx, agent, workspace, agentArgs)
}

// --- Unity AI Gateway preflight ---
//
// Kept in lockstep with Unity Gateway CLI's probe_unity_gateway_capabilities
// (https://github.com/databricks/unity-gateway/blob/main/src/ucode/databricks.py)
// Probe the Unity Catalog model-services API (v3) first, paging through it, then
// fall back to the legacy AI Gateway endpoints API (v2). A path counts only when
// its JSON body actually lists a usable resource — a 200 with an empty collection
// is "reachable but empty", not "enabled". We fail fast with the same actionable
// guidance Unity Gateway CLI surfaces so the shim's preflight matches what Unity
// Gateway CLI itself checks.

const (
	legacyEndpointsPath       = "/api/ai-gateway/v2/endpoints"
	modelServiceProbePageSize = 50
	modelServiceProbeMaxPages = 20
	modelServiceProbeMaxItems = modelServiceProbePageSize * modelServiceProbeMaxPages
	aiGatewayDocsURL          = "https://docs.databricks.com/aws/en/ai-gateway/overview-beta"
)

type gatewayProbe struct {
	reachable bool
	err       error
	// true if the API returned at least one usable resource
	resourceAvailable bool
}

func probeAIGateway(ctx context.Context, wsclient *databricks.WorkspaceClient) error {
	host := strings.TrimRight(wsclient.Config.Host, "/")

	// build a client with retries disabled so a transient 429/503 fails fast
	// instead of looping on the SDK's default 5-minute retry budget before the
	// slow first-run bootstrap even starts.
	apiClient, err := newProbeAPIClient(wsclient.Config)
	if err != nil {
		// this probe is best effort, so don't block
		return nil
	}

	modelSvc := probeModelServices(ctx, apiClient)
	// A 401 (or a 400 "invalid token") can't be rescued by trying another API,
	// so surface it before the fallback probe.
	if !modelSvc.reachable && looksLikeDefinitiveAuthFailure(modelSvc.err) {
		return aiGatewayAuthError(host, modelSvc.err.Error())
	}
	if modelSvc.resourceAvailable {
		return nil
	}

	legacy := probeLegacyEndpoints(ctx, apiClient)
	switch {
	case legacy.reachable:
		// The legacy endpoints API answered, so the gateway is enabled even if no
		// model services are visible to this caller.
		if modelSvc.err != nil {
			log.Warnf(ctx, "Unity AI Gateway model service check: %s", modelSvc.err)
		}
		return nil
	case modelSvc.reachable:
		// v3 answered; give it the benefit of the doubt rather than blocking on an
		// unconfirmed "empty".
		return nil
	case looksLikeDefinitiveAuthFailure(legacy.err):
		return aiGatewayAuthError(host, legacy.err.Error())
	case looksLikeScopeFailure(modelSvc.err):
		return aiGatewayScopeError(host, modelSvc.err.Error())
	case looksLikeScopeFailure(legacy.err):
		return aiGatewayScopeError(host, legacy.err.Error())
	case looksLikeTransient(modelSvc.err) || looksLikeTransient(legacy.err):
		// A rate-limit/5xx/network blip is not "disabled" — tell the user to retry
		// rather than sending them to the enablement docs.
		return versionNeutralGatewayError(
			fmt.Sprintf("could not verify the Databricks Unity AI Gateway on %s: the probe hit a transient error (model services: %s; legacy endpoints: %s). Retry in a moment", host, modelSvc.err.Error(), legacy.err.Error()),
		)
	case errors.Is(modelSvc.err, apierr.ErrPermissionDenied):
		return versionNeutralGatewayError(
			fmt.Sprintf("model service access could not be verified on %s (%s). The legacy endpoint fallback also failed (%s). The model service probe requires permission to list Unity Catalog model services. Verify USE CATALOG on `system`, and USE SCHEMA and EXECUTE on `system.ai`", host, modelSvc.err.Error(), legacy.err.Error()),
		)
	case errors.Is(legacy.err, apierr.ErrPermissionDenied):
		return versionNeutralGatewayError(
			fmt.Sprintf("legacy endpoint access could not be verified on %s (%s). The model service probe also failed (%s). Verify the caller's workspace permissions for the legacy endpoints listing", host, legacy.err.Error(), modelSvc.err.Error()),
		)
	default:
		return versionNeutralGatewayError(
			fmt.Sprintf("the Databricks Unity AI Gateway is not enabled on this workspace (%s): neither model services (%s) nor legacy endpoints (%s) are available. See %s", host, modelSvc.err.Error(), legacy.err.Error(), aiGatewayDocsURL),
		)
	}
}

// newProbeAPIClient builds a low-level client from cfg with the SDK's retries
// turned off, so the gateway preflight is one-shot. Clearing ErrorRetriable and
// TransientErrors makes a retriable status (429, 503, ...) surface on the first
// attempt rather than looping until the retry budget expires; it mirrors
// client.New, which builds the retrying client the rest of the CLI uses.
func newProbeAPIClient(cfg *config.Config) (*client.DatabricksClient, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	clientCfg.ErrorRetriable = func(context.Context, error) bool { return false }
	clientCfg.TransientErrors = nil
	return client.NewWithClient(cfg, httpclient.NewApiClient(clientCfg))
}

func probeModelServices(ctx context.Context, apiClient *client.DatabricksClient) gatewayProbe {
	it := catalog.NewAiGateway(apiClient).ListModelServices(ctx, catalog.ListModelServicesRequest{PageSize: modelServiceProbePageSize})
	_, err := it.Next(ctx)
	if err != nil {
		if err == listing.ErrNoMoreItems {
			// endpoint is reachable but there are no results
			return gatewayProbe{reachable: true}
		}
		return gatewayProbe{err: err}
	}
	return gatewayProbe{reachable: true, resourceAvailable: true}
}

type legacyEndpointsResponse struct {
	Endpoints []any `json:"endpoints"`
}

func probeLegacyEndpoints(ctx context.Context, apiClient *client.DatabricksClient) gatewayProbe {
	var resp legacyEndpointsResponse
	err := apiClient.Do(ctx, http.MethodGet, legacyEndpointsPath, auth.WorkspaceIDHeaders(apiClient.Config), map[string]any{
		"page_size": "1",
	}, nil, &resp)
	if err != nil {
		return gatewayProbe{err: err}
	}
	if len(resp.Endpoints) > 0 {
		return gatewayProbe{reachable: true, resourceAvailable: true}
	}
	return gatewayProbe{reachable: true}
}

var (
	gatewayDetailV3 = regexp.MustCompile(`(?i)\bv3\b`)
	gatewayDetailV2 = regexp.MustCompile(`(?i)\bv2\b`)
)

// rewrite the internal v2/v3 API labels in a failure reason into user-facing terms
func versionNeutralGatewayError(reason string) error {
	reason = gatewayDetailV3.ReplaceAllString(reason, "model service")
	reason = gatewayDetailV2.ReplaceAllString(reason, "legacy endpoint")
	return errors.New(reason)
}

// returns true when retrying another workspace API can't rescue the token.
// A bare 403 is left out: it can be endpoint-specific authorization, so the
// preflight still tries the fallback before giving up.
func looksLikeDefinitiveAuthFailure(err error) bool {
	// HTTP 401
	if errors.Is(err, apierr.ErrUnauthenticated) {
		return true
	}
	return errors.Is(err, apierr.ErrBadRequest) && strings.Contains(strings.ToLower(err.Error()), "invalid token")
}

// matches a 403 that reports the OAuth token is missing a required scope (which
// re-login can fix), as opposed to a plain permission 403.
func looksLikeScopeFailure(err error) bool {
	reason := strings.ToLower(err.Error())
	return errors.Is(err, apierr.ErrPermissionDenied) && strings.Contains(reason, "oauth token") && strings.Contains(reason, "required scopes")
}

func looksLikePermissionFailure(reason string) bool {
	return strings.Contains(reason, "HTTP 403")
}

func looksLikeTransient(err error) bool {
	return strings.HasPrefix(err.Error(), "network error") ||
		errors.Is(err, apierr.ErrTooManyRequests) || // HTTP 429
		errors.Is(err, apierr.ErrInternalError) || // HTTP 500
		errors.Is(err, apierr.ErrTemporarilyUnavailable) || // HTTP 503
		errors.Is(err, apierr.ErrDeadlineExceeded) // HTTP 504
}

func aiGatewayAuthError(host, reason string) error {
	return versionNeutralGatewayError(
		fmt.Sprintf("the Databricks workspace %s rejected the access token (%s). Try:\n  databricks auth logout --host %s\n  databricks auth login --host %s", host, reason, host, host),
	)
}

func aiGatewayScopeError(host, reason string) error {
	return versionNeutralGatewayError(
		fmt.Sprintf("the access token for %s is missing an OAuth scope required by the AI Gateway APIs (%s). Re-authenticate to mint a token with the needed scopes:\n  databricks auth login --host %s", host, reason, host),
	)
}

func hasNonEmptyCollection(payload any, key string) bool {
	obj, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	arr, ok := obj[key].([]any)
	return ok && len(arr) > 0
}

func stringField(payload any, key string) string {
	obj, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := obj[key].(string)
	return s
}

// --- end Unity AI Gateway preflight ---

func bootstrapAndLaunchAgent(ctx context.Context, agent agentSpec, workspace string, agentArgs []string) error {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}

	// Reconstruct the PATH tooling runs under on every launch rather than caching
	// it: every entry is a known location, so deriving it can't restore a stale
	// path (e.g. a versioned CLI dir from an older session). Drop the shim dir so
	// Unity Gateway CLI execs the real agent rather than this wrapper, then prepend
	// uv/ug's bin, this databricks CLI's own dir (so it's used by ug/the spawned agent),
	// and the Node bin ensureNode installs into.
	removePath(ctx, filepath.Join(home, binDir))
	prependPath(ctx, filepath.Join(home, ".local", "bin")) // uv
	if self, err := os.Executable(); err == nil {
		prependPath(ctx, filepath.Dir(self))
	}
	prependPath(ctx, filepath.Join(home, depsDir, "node", "bin"))

	if err := ensureToolchain(ctx, home); err != nil {
		return err
	}

	// npm is on PATH now (shipped or just installed); silence its update-notifier
	// box so it doesn't clutter the agent session.
	disableNpmUpdateNotifier(ctx)

	// Put npm's global bin on PATH so an npm-installed agent binary resolves after
	// Unity Gateway CLI installs it.
	if prefix := npmGlobalPrefix(ctx); prefix != "" {
		prependPath(ctx, filepath.Join(prefix, "bin"))
	}

	return launchAgent(ctx, home, agent, workspace, agentArgs)
}

func toolchainReady() bool {
	_, ugErr := exec.LookPath("ucode")
	_, npmErr := exec.LookPath("npm")
	return ugErr == nil && npmErr == nil
}

func ensureToolchain(ctx context.Context, home string) error {
	if toolchainReady() {
		return nil
	}
	unlock, err := acquireSetupLock(ctx, home)
	if err != nil {
		return err
	}
	defer unlock()
	// Another client may have finished the install while we waited for the lock.
	if toolchainReady() {
		return nil
	}

	// 1. uv (installs into ~/.local/bin).
	if _, err := exec.LookPath("uv"); err != nil {
		cmdio.LogString(ctx, "Installing uv...")
		if err := runShell(ctx, "curl -LsSf https://astral.sh/uv/install.sh | sh"); err != nil {
			return fmt.Errorf("failed to install uv: %w", err)
		}
	}

	// 2. Unity Gateway CLI (pinned stock upstream release).
	if _, err := exec.LookPath("ucode"); err != nil {
		cmdio.LogString(ctx, "Installing Unity Gateway CLI...")
		if err := runCommand(ctx, "uv", "tool", "install", "git+https://github.com/"+ugRepo+"@"+ugVersion); err != nil {
			return fmt.Errorf("failed to install Unity Gateway CLI: %w", err)
		}
	}

	// 3. Node/npm (installs into depsDir/node/bin)
	if _, err := exec.LookPath("npm"); err != nil {
		cmdio.LogString(ctx, "Installing npm...")
		if _, err := ensureNode(ctx, home); err != nil {
			return err
		}
	}

	return nil
}

// take a machine-local lock (an O_EXCL sentinel file) serializing first-run setup
// across SSH clients. It blocks until the lock is free, reclaiming one left behind
// by a dead process after setupLockStaleAfter. The returned func releases the lock.
func acquireSetupLock(ctx context.Context, home string) (func(), error) {
	lockPath := filepath.Join(home, agentDir, setupLockName)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create %s: %w", filepath.Dir(lockPath), err)
	}
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("failed to acquire setup lock: %w", err)
		}
		// Held by another process: reclaim it if stale, otherwise wait and retry.
		if info, statErr := os.Stat(lockPath); statErr == nil && time.Since(info.ModTime()) > setupLockStaleAfter {
			log.Warnf(ctx, "reclaiming stale agent-shim setup lock at %s", lockPath)
			_ = os.Remove(lockPath)
			continue
		}
		log.Infof(ctx, "waiting for a concurrent setup to finish")
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func launchAgent(ctx context.Context, home string, agent agentSpec, workspace string, agentArgs []string) error {
	ugPath, err := exec.LookPath("ucode")
	if err != nil {
		return fmt.Errorf("the Unity Gateway CLI was not found on PATH after setup: %w", err)
	}
	contextArgs, err := injectAgentContext(ctx, home, agent)
	if err != nil {
		return err
	}
	argv := []string{"ucode", agent.name}
	if workspace != "" {
		argv = append(argv, "--workspace", workspace)
	}
	argv = append(argv, "--")
	argv = append(argv, contextArgs...)
	argv = append(argv, agentArgs...)
	// Pass the session token as DATABRICKS_BEARER so Unity Gateway CLI authenticates headlessly.
	if env.Get(ctx, "DATABRICKS_BEARER") == "" {
		if token := env.Get(ctx, "DATABRICKS_TOKEN"); token != "" {
			_ = os.Setenv("DATABRICKS_BEARER", token)
		}
	}
	return execProcess(ugPath, argv, os.Environ())
}

// put the Databricks context where the agent reads it, returning any extra argv.
func injectAgentContext(ctx context.Context, home string, agent agentSpec) ([]string, error) {
	systemContext := agentSystemContext(env.Get(ctx, workspaceHomeEnv))
	switch {
	case agent.contextFlag != "":
		// Keep the scratch context file under the shim's own directory so it never
		// clobbers an unrelated file the user happens to have in their home.
		dir := filepath.Join(home, agentDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", dir, err)
		}
		f := filepath.Join(dir, contextFile)
		if err := os.WriteFile(f, []byte(systemContext), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write agent context file: %w", err)
		}
		return []string{agent.contextFlag, f}, nil
	case agent.contextHomeFile != "":
		f := filepath.Join(home, agent.contextHomeFile)
		if _, err := os.Stat(f); err == nil {
			return nil, nil // already present — don't clobber
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to stat %s: %w", f, err)
		}
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			return nil, fmt.Errorf("failed to create %s: %w", filepath.Dir(f), err)
		}
		if err := os.WriteFile(f, []byte(systemContext), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", f, err)
		}
		return nil, nil
	default:
		return nil, nil
	}
}

// download the latest Krypton LTS Node into deps/node once, returning its bin.
// Linux-only by design: the shim runs on the serverless driver, so the tarball
// name is hardcoded to linux while nodeDownloadArch guards the arch.
func ensureNode(ctx context.Context, home string) (string, error) {
	depsRoot := filepath.Join(home, depsDir)
	nodeDir := filepath.Join(depsRoot, "node")
	nodeBin := filepath.Join(nodeDir, "bin")
	if _, err := os.Stat(filepath.Join(nodeBin, "npm")); err == nil {
		return nodeBin, nil
	}

	arch := nodeDownloadArch(runtime.GOARCH)
	if arch == "" {
		return "", fmt.Errorf("unsupported architecture for Node download: %s", runtime.GOARCH)
	}
	const base = "https://nodejs.org/dist/latest-krypton"
	tarName, wantSum, err := latestNodeTarball(ctx, base+"/SHASUMS256.txt", arch)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(depsRoot, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", depsRoot, err)
	}
	// Download to a temp file, verifying its SHA256 from SHASUMS256.txt before use.
	tarball, err := downloadVerified(ctx, base+"/"+tarName, wantSum)
	if err != nil {
		return "", err
	}
	defer os.Remove(tarball)
	// Extract into a sibling temp dir and atomically rename it into place, so an
	// interrupted extraction never leaves a half-populated node dir that the
	// os.Stat(npm) check above would then wrongly accept as a finished install.
	tmpDir, err := os.MkdirTemp(depsRoot, "node-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) // no-op once renamed; cleans up a failed extraction
	// Node's .tar.xz is the smallest download; extract it with the system tar.
	if err := runCommand(ctx, "tar", "-xJf", tarball, "--strip-components=1", "-C", tmpDir); err != nil {
		return "", fmt.Errorf("failed to extract Node.js: %w", err)
	}
	// Clear any partial leftover from a previously-interrupted run, then publish
	// atomically (same-filesystem rename, since tmpDir is a sibling of nodeDir).
	if err := os.RemoveAll(nodeDir); err != nil {
		return "", fmt.Errorf("failed to remove %s: %w", nodeDir, err)
	}
	if err := os.Rename(tmpDir, nodeDir); err != nil {
		return "", fmt.Errorf("failed to install Node.js: %w", err)
	}
	return nodeBin, nil
}

// fetch url to a temp file, failing unless its SHA256 matches wantSum. The caller
// is responsible for removing the returned file.
func downloadVerified(ctx context.Context, url, wantSum string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.CreateTemp("", "node-*.tar.xz")
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, sum), resp.Body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, wantSum) {
		os.Remove(f.Name())
		return "", fmt.Errorf("checksum mismatch for %s: got %s, want %s", url, got, wantSum)
	}
	return f.Name(), nil
}

func nodeDownloadArch(goarch string) string {
	switch goarch {
	case "amd64":
		return "x64"
	case "arm64":
		return "arm64"
	default:
		return ""
	}
}

func latestNodeTarball(ctx context.Context, shasumsURL, arch string) (name, sum string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, shasumsURL, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to fetch Node checksums: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", fmt.Errorf("failed to read Node checksums: %w", err)
	}
	re := regexp.MustCompile(`^([0-9a-f]{64})\s+(node-v[0-9.]+-linux-` + regexp.QuoteMeta(arch) + `\.tar\.xz)$`)
	for line := range strings.SplitSeq(string(body), "\n") {
		if m := re.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			return m[2], m[1], nil
		}
	}
	return "", "", fmt.Errorf("no linux-%s Node tarball found in %s", arch, shasumsURL)
}

// turn off npm's "new version available" box so it doesn't clutter the installation
// progress. Best-effort: it's cosmetic, so a failure (e.g. npm not runnable) is
// logged and ignored rather than blocking the launch.
func disableNpmUpdateNotifier(ctx context.Context) {
	if err := exec.CommandContext(ctx, "npm", "config", "set", "update-notifier", "false").Run(); err != nil {
		log.Debugf(ctx, "failed to disable npm update-notifier: %v", err)
	}
}

func npmGlobalPrefix(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "npm", "prefix", "-g").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runShell(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func prependPath(ctx context.Context, dir string) {
	dirs := []string{dir}
	for _, d := range filepath.SplitList(env.Get(ctx, "PATH")) {
		if d != dir {
			dirs = append(dirs, d)
		}
	}
	_ = os.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

func removePath(ctx context.Context, dir string) {
	var dirs []string
	for _, d := range filepath.SplitList(env.Get(ctx, "PATH")) {
		if d != dir {
			dirs = append(dirs, d)
		}
	}
	_ = os.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

func agentSystemContext(wsHome string) string {
	cwd := "the user's Databricks workspace home directory"
	if wsHome != "" {
		cwd = wsHome
	}
	return fmt.Sprintf(`You are running inside a "databricks ssh connect" session on the driver node of a
Databricks serverless cluster.
- The "databricks" CLI is installed and already authenticated: DATABRICKS_HOST and DATABRICKS_TOKEN are set in the environment, so "databricks ..." commands work with no "databricks auth login". The same token governs Unity Catalog and serving-endpoint access.
- This container is ephemeral; only paths under /Workspace and /Volumes persist across sessions. Your working directory is %s.
- You can run shell commands and use the "databricks" CLI to explore the workspace.`, cwd)
}
