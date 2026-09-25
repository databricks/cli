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
	agentRootDir = ".agent-shim"

	// directory which holds the per-agent wrappers; must match remoteShimDir in client.go.
	agentBinDir = agentRootDir + "/bin"

	// directory which holds toolchain deps (npm, uv) fetched when the image ships none.
	agentDepsDir = agentRootDir + "/deps"

	// the GitHub repo the shim installs the Unity Gateway CLI from.
	ugRepo = "databricks/unity-gateway"

	// pins the Unity Gateway CLI commit the shim installs.
	ugCommit = "7f408803f9c0fe4958e6d264fd94595283a49a19" // v0.1.0

	// pins the version of uv downloaded
	uvBaseURL = "https://github.com/astral-sh/uv/releases/download/0.12.18/"

	// SHA-256 of each pinned uv release tarball
	uvX64LinuxChecksum   = "89eadd7c76fc063887959510d5ba0ab1264dfd5f1143b925ddb73021a40acf16"
	uvArm64LinuxChecksum = "afb6291f3f0a6b4521fc67b947822506c41dde5b60d2189dd8f3695b2ac8c9e7"

	// overrides for uv tool installs so they are isolated to a directory we control
	uvToolDir    = agentDepsDir + "/uv_tools"
	uvToolBinDir = agentDepsDir + "/uv_tools_bin"

	// pins the version of node/npm downloaded
	nodeVersion = "v24.21.0"
	nodeBaseURL = "https://nodejs.org/dist/" + nodeVersion + "/"

	// SHA-256 of each pinned node release tarball
	nodeX64LinuxChecksum   = "fd8e59d5a511510f6a298afb548f18c7d2b1be404d8b4a27d94fbe49f56cb2d6"
	nodeArm64LinuxChecksum = "6ad1325edbdb5649c379b75a237147a666c95d4f9ae8d340fef2d1575d289ad2"

	// environment variable used to pass the user's workspace home to the shim for the agent context.
	workspaceHomeEnv = "DATABRICKS_WORKSPACE_HOME"

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
		log.Debugf(ctx, "Skipping Unity AI Gateway check: %s", err)
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

// --- end Unity AI Gateway preflight ---

func bootstrapAndLaunchAgent(ctx context.Context, agent agentSpec, workspace string, agentArgs []string) error {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}

	// Reconstruct the PATH tooling runs under on every launch rather than caching
	// it: every entry is a known location, so deriving it can't restore a stale
	// path (e.g. a versioned CLI dir from an older session). Drop the shim dir so
	// Unity Gateway CLI execs the real agent rather than this wrapper
	removePath(ctx, filepath.Join(home, agentBinDir))
	// add this CLI to PATH so agents can use it
	if self, err := os.Executable(); err == nil {
		prependPath(ctx, filepath.Dir(self))
	}

	err = ensureToolchain(ctx, home)
	if err != nil {
		return err
	}

	// npm is on PATH now (shipped or just installed); silence its update-notifier
	// box so it doesn't clutter the agent session.
	disableNpmUpdateNotifier(ctx)

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
		return nil
	}
	defer unlock()
	// Another client may have finished the install while we waited for the lock.
	if toolchainReady() {
		return nil
	}

	uvToolDirAbsolute := filepath.Join(home, uvToolDir)
	uvToolBinDirAbsolute := filepath.Join(home, uvToolBinDir)

	if _, err := exec.LookPath("uv"); err != nil {
		cmdio.LogString(ctx, "Installing uv...")
		uvPath, err := ensureBinary(ctx, home, uvArchiveSpec(runtime.GOARCH))
		if err != nil {
			return fmt.Errorf("failed to install uv: %w", err)
		}
		prependPath(ctx, uvPath, uvToolBinDirAbsolute)
	}

	if _, err := exec.LookPath("ucode"); err != nil {
		cmdio.LogString(ctx, "Installing Unity Gateway CLI...")
		env := os.Environ()
		env = append(env, "UV_TOOL_DIR="+uvToolDirAbsolute, "UV_TOOL_BIN_DIR="+uvToolBinDirAbsolute)
		if err := runCommand(ctx, env, "uv", "tool", "install", "git+https://github.com/"+ugRepo+"@"+ugCommit); err != nil {
			return fmt.Errorf("failed to install Unity Gateway CLI: %w", err)
		}
	}

	if _, err := exec.LookPath("npm"); err != nil {
		cmdio.LogString(ctx, "Installing npm...")
		nodePath, err := ensureBinary(ctx, home, nodeArchiveSpec(runtime.GOARCH))
		if err != nil {
			return err
		}
		prependPath(ctx, nodePath)
	}

	return nil
}

type archiveSpec struct {
	// "" if this is an unsupported architecture
	archiveURL string
	binaryName string
	dirName    string
	// sub-directory inside dirName where the binary is located, "" if it's the same as dirName
	binDirName string
	checksum   string
}

// ensure the specified binary is installed, installing it if it's not already
// present. Returns the path to the directory containing the installed binary
func ensureBinary(ctx context.Context, home string, spec archiveSpec) (string, error) {
	if spec.archiveURL == "" {
		return "", fmt.Errorf("unsupported architecture for %s download: %s", spec.binaryName, runtime.GOARCH)
	}

	depsRoot := filepath.Join(home, agentDepsDir)
	depsDir := filepath.Join(depsRoot, spec.dirName)
	binDir := depsDir
	if spec.binDirName != "" {
		binDir = filepath.Join(depsDir, spec.binDirName)
	}
	if _, err := os.Stat(filepath.Join(binDir, spec.binaryName)); err == nil {
		return binDir, nil
	}

	if err := os.MkdirAll(depsRoot, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", depsRoot, err)
	}

	tarball, err := downloadVerified(ctx, spec)
	if err != nil {
		return "", err
	}
	defer os.Remove(tarball)
	// Extract into a sibling temp dir and atomically rename it into place, so an
	// interrupted extraction never leaves a half-populated dir that the os.Stat
	// check above would then wrongly accept as a finished install.
	tmpDir, err := os.MkdirTemp(depsRoot, spec.binaryName+"-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir) // no-op once renamed; cleans up a failed extraction
	// extract with tar for brevity
	if err := runCommand(ctx, nil, "tar", "-xf", tarball, "--strip-components=1", "-C", tmpDir); err != nil {
		return "", fmt.Errorf("failed to extract %s: %w", spec.binaryName, err)
	}
	// Clear any partial leftover from a previously-interrupted run, then publish
	// atomically (same-filesystem rename, since tmpDir is a sibling of nodeDir).
	if err := os.RemoveAll(depsDir); err != nil {
		return "", fmt.Errorf("failed to remove %s: %w", depsDir, err)
	}
	if err := os.Rename(tmpDir, depsDir); err != nil {
		return "", fmt.Errorf("failed to install %s: %w", spec.binaryName, err)
	}
	return binDir, nil
}

func uvArchiveSpec(goarch string) archiveSpec {
	spec := archiveSpec{dirName: "uv", binaryName: "uv"}
	switch goarch {
	case "amd64":
		spec.archiveURL = uvBaseURL + "uv-x86_64-unknown-linux-gnu.tar.gz"
		spec.checksum = uvX64LinuxChecksum
	case "arm64":
		spec.archiveURL = uvBaseURL + "uv-aarch64-unknown-linux-gnu.tar.gz"
		spec.checksum = uvArm64LinuxChecksum
	}
	return spec
}

// take a machine-local lock (an O_EXCL sentinel file) serializing first-run setup
// across SSH clients. It blocks until the lock is free, reclaiming one left behind
// by a dead process after setupLockStaleAfter. The returned func releases the lock.
func acquireSetupLock(ctx context.Context, home string) (func(), error) {
	lockPath := filepath.Join(home, agentRootDir, setupLockName)
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
		dir := filepath.Join(home, agentRootDir)
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

// fetch url to a temp file, failing unless its SHA256 matches the expected checksum.
// The caller is responsible for removing the returned file.
func downloadVerified(ctx context.Context, spec archiveSpec) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, spec.archiveURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", spec.archiveURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download %s: HTTP %d", spec.archiveURL, resp.StatusCode)
	}

	// either .tar.gz or .tar.xz
	fileExt := spec.archiveURL[len(spec.archiveURL)-7:]
	f, err := os.CreateTemp("", spec.binaryName+"-*."+fileExt)
	if err != nil {
		return "", err
	}
	sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, sum), resp.Body); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("failed to download %s: %w", spec.archiveURL, err)
	}
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); !strings.EqualFold(got, spec.checksum) {
		os.Remove(f.Name())
		return "", fmt.Errorf("checksum mismatch for %s: got %s, want %s", spec.archiveURL, got, spec.checksum)
	}
	return f.Name(), nil
}

func nodeArchiveSpec(goarch string) archiveSpec {
	spec := archiveSpec{binDirName: "bin", dirName: "node", binaryName: "npm"}
	switch goarch {
	case "amd64":
		spec.archiveURL = nodeBaseURL + "node-" + nodeVersion + "-linux-x64.tar.xz"
		spec.checksum = nodeX64LinuxChecksum
	case "arm64":
		spec.archiveURL = nodeBaseURL + "node-" + nodeVersion + "-linux-arm64.tar.xz"
		spec.checksum = nodeArm64LinuxChecksum
	}
	return spec
}

// turn off npm's "new version available" box so it doesn't clutter the installation
// progress. Best-effort: it's cosmetic, so a failure (e.g. npm not runnable) is
// logged and ignored rather than blocking the launch.
func disableNpmUpdateNotifier(ctx context.Context) {
	if err := exec.CommandContext(ctx, "npm", "config", "set", "update-notifier", "false").Run(); err != nil {
		log.Debugf(ctx, "failed to disable npm update-notifier: %v", err)
	}
}

func runCommand(ctx context.Context, env []string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if len(env) > 0 {
		cmd.Env = env
	}
	return cmd.Run()
}

func prependPath(ctx context.Context, dirs ...string) {
	for _, d := range filepath.SplitList(env.Get(ctx, "PATH")) {
		match := false
		for _, toAdd := range dirs {
			match = match || d == toAdd
		}
		if !match {
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
