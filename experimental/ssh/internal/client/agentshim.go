package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

const (
	// agentDir (home-relative) is the shim's working area.
	agentDir = ".agent-shim"

	// shimDir holds the per-agent wrappers; must match remoteShimDir in client.go.
	shimDir = agentDir + "/bin"

	// ucodeRepo is the GitHub repo the shim installs ucode from (latest release).
	ucodeRepo = "databricks/ucode"

	// workspaceHomeEnv passes the user's workspace home to the shim for the agent context.
	workspaceHomeEnv = "DATABRICKS_WORKSPACE_HOME"

	// depsDir (home-relative) holds toolchain deps (Node/npm) fetched when the image ships none.
	depsDir = agentDir + "/deps"

	// contextFile is the scratch file holding the Databricks session context.
	contextFile = "agent-system-context.md"

	// readyFile marks that first-run toolchain setup completed.
	readyFile = ".ready"
)

type agentSpec struct {
	name string
	// contextFlag passes the context file via this CLI flag (Claude's --append-system-prompt-file).
	contextFlag string
	// contextHomeFile writes the context to this $HOME instructions file (codex's AGENTS.md).
	contextHomeFile string
}

// supportedAgents are the agents the shim launches — limited to those whose ucode
// command accepts --workspace (how we target the workspace headlessly).
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
	// ucode targets this workspace via --workspace, so it configures without prompting.
	workspace := strings.TrimRight(client.Config.Host, "/")
	return bootstrapAndLaunchAgent(ctx, agent, workspace, agentArgs)
}

// --- Unity AI Gateway preflight ---
//
// Kept in lockstep with ucode's probe_unity_gateway_capabilities
// (databricks/ucode, src/ucode/databricks.py): probe the Unity Catalog
// model-services API (v3) first, paging through it, then fall back to the legacy
// AI Gateway endpoints API (v2). A path counts only when its JSON body actually
// lists a usable resource — a 200 with an empty collection is "reachable but
// empty", not "enabled". We fail fast with the same actionable guidance ucode
// surfaces so the shim's preflight matches what ucode itself checks.

const (
	modelServicesPath         = "/api/2.1/unity-catalog/model-services"
	legacyEndpointsPath       = "/api/ai-gateway/v2/endpoints"
	modelServiceProbePageSize = 50
	modelServiceProbeMaxPages = 20
	aiGatewayDocsURL          = "https://docs.databricks.com/aws/en/ai-gateway/overview-beta"

	// modelServiceEmptyDetail matches ucode's wording for a reachable model-services
	// API that lists nothing the caller can use — almost always a UC grant gap.
	modelServiceEmptyDetail = "reachable, no accessible model services returned; " +
		"check USE CATALOG on system, and USE SCHEMA and EXECUTE on system.ai"
)

// gatewayProbe is the outcome of probing one AI Gateway API. resourceAvailable
// means the API returned at least one usable resource; conclusive is false when
// paging couldn't be completed, so "no resources" was never actually confirmed.
type gatewayProbe struct {
	reachable         bool
	detail            string
	resourceAvailable bool
	conclusive        bool
}

// probeAIGateway fails fast if the workspace's Unity AI Gateway can't run an
// agent. It returns nil when the gateway is usable — logging a warning when it's
// reachable but exposes no model services to this caller — and an actionable
// error otherwise.
func probeAIGateway(ctx context.Context, client *databricks.WorkspaceClient) error {
	host := strings.TrimRight(client.Config.Host, "/")

	modelSvc := probeModelServices(ctx, client, host)
	// A 401 (or a 400 "invalid token") can't be rescued by trying another API,
	// so surface it before the fallback probe.
	if !modelSvc.reachable && looksLikeDefinitiveAuthFailure(modelSvc.detail) {
		return aiGatewayAuthError(host, modelSvc.detail)
	}
	if modelSvc.resourceAvailable {
		return nil
	}

	legacy := probeLegacyEndpoints(ctx, client, host)
	switch {
	case legacy.reachable:
		// The legacy endpoints API answered, so the gateway is enabled even if no
		// model services are visible to this caller.
		log.Warnf(ctx, "Unity AI Gateway model service check: %s", modelSvc.detail)
		return nil
	case modelSvc.reachable && !modelSvc.conclusive:
		// v3 answered but couldn't be paged to completion; give it the benefit of
		// the doubt rather than blocking on an unconfirmed "empty".
		return nil
	case looksLikeDefinitiveAuthFailure(legacy.detail):
		return aiGatewayAuthError(host, legacy.detail)
	case looksLikeScopeFailure(modelSvc.detail):
		return aiGatewayScopeError(host, modelSvc.detail)
	case looksLikeScopeFailure(legacy.detail):
		return aiGatewayScopeError(host, legacy.detail)
	case looksLikePermissionFailure(modelSvc.detail):
		return fmt.Errorf("Databricks Unity AI Gateway model service access could not be verified on %s (%s). The legacy endpoint fallback also failed (%s). The model service probe requires permission to list Unity Catalog model services. Verify USE CATALOG on `system`, and USE SCHEMA and EXECUTE on `system.ai`", host, modelSvc.detail, legacy.detail)
	case looksLikePermissionFailure(legacy.detail):
		return fmt.Errorf("Databricks Unity AI Gateway legacy endpoint access could not be verified on %s (%s). The model service probe also failed (%s). Verify the caller's workspace permissions for the legacy endpoints listing", host, legacy.detail, modelSvc.detail)
	default:
		return fmt.Errorf("Databricks Unity AI Gateway is not enabled on this workspace (%s): neither model services (%s) nor legacy endpoints (%s) are available. See %s", host, modelSvc.detail, legacy.detail, aiGatewayDocsURL)
	}
}

// probeModelServices probes the UC model-services API (v3), paging until it finds
// an accessible model service, exhausts the cursor, or hits the page cap.
func probeModelServices(ctx context.Context, client *databricks.WorkspaceClient, host string) gatewayProbe {
	pageToken := ""
	for page := 0; page < modelServiceProbeMaxPages; page++ {
		reqURL := fmt.Sprintf("%s%s?page_size=%d", host, modelServicesPath, modelServiceProbePageSize)
		if pageToken != "" {
			reqURL += "&page_token=" + url.QueryEscape(pageToken)
		}
		payload, reason := gatewayGetJSON(ctx, client, reqURL)
		if payload == nil {
			// A first-page failure is the real reachability signal; a later page
			// failing still means the API answered at least once, so treat it as
			// reachable-but-inconclusive rather than a confirmed "empty".
			if page == 0 {
				return gatewayProbe{detail: versionNeutralGatewayDetail(reason), conclusive: true}
			}
			return gatewayProbe{reachable: true, detail: "reachable"}
		}
		if hasNonEmptyCollection(payload, "model_services") {
			return gatewayProbe{reachable: true, detail: "reachable, accessible model service returned", resourceAvailable: true, conclusive: true}
		}
		pageToken = stringField(payload, "next_page_token")
		if pageToken == "" {
			return gatewayProbe{reachable: true, detail: modelServiceEmptyDetail, conclusive: true}
		}
	}
	return gatewayProbe{reachable: true, detail: "reachable"}
}

// probeLegacyEndpoints probes the legacy AI Gateway endpoints API (v2).
func probeLegacyEndpoints(ctx context.Context, client *databricks.WorkspaceClient, host string) gatewayProbe {
	payload, reason := gatewayGetJSON(ctx, client, host+legacyEndpointsPath+"?page_size=1")
	if payload == nil {
		return gatewayProbe{detail: versionNeutralGatewayDetail(reason), conclusive: true}
	}
	if hasNonEmptyCollection(payload, "endpoints") {
		return gatewayProbe{reachable: true, detail: "reachable, accessible endpoint returned", resourceAvailable: true, conclusive: true}
	}
	return gatewayProbe{reachable: true, detail: "reachable, no accessible endpoints returned", conclusive: true}
}

// gatewayGetJSON issues an authenticated GET expecting JSON. It returns the
// decoded payload on HTTP 200, or (nil, reason) on any failure — mirroring
// ucode's _http_get_json, including a short response-body excerpt so gateway auth
// errors (e.g. a 400 whose body is "Invalid Token") stay visible in the reason.
func gatewayGetJSON(ctx context.Context, client *databricks.WorkspaceClient, reqURL string) (any, string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Sprintf("network error: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	if err := client.Config.Authenticate(req); err != nil {
		return nil, fmt.Sprintf("network error: %v", err)
	}
	resp, err := (&http.Client{Transport: client.Config.HTTPTransport, Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Sprintf("network error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusOK {
		var payload any
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Sprintf("response was not valid JSON (%v)", err)
		}
		return payload, ""
	}
	reason := fmt.Sprintf("HTTP %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	if excerpt := strings.TrimSpace(string(body)); excerpt != "" {
		if len(excerpt) > 200 {
			excerpt = excerpt[:200]
		}
		reason += ": " + excerpt
	}
	return nil, reason
}

var (
	gatewayDetailV3 = regexp.MustCompile(`(?i)\bv3\b`)
	gatewayDetailV2 = regexp.MustCompile(`(?i)\bv2\b`)
)

// versionNeutralGatewayDetail rewrites the internal v2/v3 API labels in a failure
// reason into user-facing terms, matching ucode so the error text stays in sync.
func versionNeutralGatewayDetail(detail string) string {
	detail = gatewayDetailV3.ReplaceAllString(detail, "model service")
	return gatewayDetailV2.ReplaceAllString(detail, "legacy endpoint")
}

// looksLikeDefinitiveAuthFailure is true when retrying another workspace API
// can't rescue the token. A bare 403 is left out: it can be endpoint-specific
// authorization, so the preflight still tries the fallback before giving up.
func looksLikeDefinitiveAuthFailure(reason string) bool {
	if strings.Contains(reason, "HTTP 401") {
		return true
	}
	return strings.Contains(reason, "HTTP 400") && strings.Contains(strings.ToLower(reason), "invalid token")
}

// looksLikeScopeFailure matches a 403 that reports the OAuth token is missing a
// required scope (which re-login can fix), as opposed to a plain permission 403.
func looksLikeScopeFailure(reason string) bool {
	l := strings.ToLower(reason)
	return strings.Contains(l, "http 403") && strings.Contains(l, "oauth token") && strings.Contains(l, "required scopes")
}

func looksLikePermissionFailure(reason string) bool {
	return strings.Contains(reason, "HTTP 403")
}

func aiGatewayAuthError(host, reason string) error {
	return fmt.Errorf("Databricks rejected the access token for %s (%s). Try:\n  databricks auth logout --host %s\n  databricks auth login --host %s", host, reason, host, host)
}

func aiGatewayScopeError(host, reason string) error {
	return fmt.Errorf("the access token for %s is missing an OAuth scope required by the AI Gateway APIs (%s). Re-authenticate to mint a token with the needed scopes:\n  databricks auth login --host %s", host, reason, host)
}

// hasNonEmptyCollection reports whether payload is a JSON object with a non-empty
// array at key.
func hasNonEmptyCollection(payload any, key string) bool {
	obj, ok := payload.(map[string]any)
	if !ok {
		return false
	}
	arr, ok := obj[key].([]any)
	return ok && len(arr) > 0
}

// stringField returns payload[key] when payload is a JSON object holding a string
// there, else "".
func stringField(payload any, key string) string {
	obj, ok := payload.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := obj[key].(string)
	return s
}

func bootstrapAndLaunchAgent(ctx context.Context, agent agentSpec, workspace string, agentArgs []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}
	shimDir := filepath.Join(home, shimDir)
	readyFile := filepath.Join(home, agentDir, readyFile)

	// Fast path: first-run setup recorded the resolved PATH; restore it and launch.
	if data, err := os.ReadFile(readyFile); err == nil {
		_ = os.Setenv("PATH", strings.TrimSpace(string(data)))
		return launchUcodeAgent(agent, workspace, agentArgs)
	}

	// Build the PATH tooling runs under. Drop the shim dir so ucode installs and
	// execs the real agent rather than this wrapper; prepend ucode's install bin
	// and the uploaded databricks CLI's dir (so ucode finds it and skips its own
	// install).
	removePath(shimDir)
	prependPath(filepath.Join(home, ".local", "bin"))
	if self, err := os.Executable(); err == nil {
		prependPath(filepath.Dir(self))
	}

	ui := newProgressUI(ctx)

	// 1. uv (installs into ~/.local/bin).
	if _, err := exec.LookPath("uv"); err != nil {
		if err := ui.runStep("Installing dependencies", func(out io.Writer) error {
			if err := runShell(ctx, out, "curl -LsSf https://astral.sh/uv/install.sh | sh"); err != nil {
				return fmt.Errorf("failed to install uv: %w", err)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// 2. ucode (latest stock upstream release).
	if _, err := exec.LookPath("ucode"); err != nil {
		if err := ui.runStep("Installing ucode", func(out io.Writer) error {
			ref, err := latestUcodeRef(ctx, "https://api.github.com/repos/"+ucodeRepo+"/releases/latest")
			if err != nil {
				return err
			}
			if err := runCommand(ctx, out, "uv", "tool", "install", "git+https://github.com/"+ucodeRepo+"@"+ref); err != nil {
				return fmt.Errorf("failed to install ucode: %w", err)
			}
			return nil
		}); err != nil {
			return err
		}
	}

	// 3. Node/npm. Some agents (e.g. Claude Code) need it and serverless images don't ship it.
	if _, err := exec.LookPath("npm"); err != nil {
		var nodeBin string
		if err := ui.runStep("Installing "+agent.name+" dependencies", func(out io.Writer) error {
			bin, err := ensureNode(ctx, home, out)
			nodeBin = bin
			return err
		}); err != nil {
			return err
		}
		prependPath(nodeBin)
	}

	// npm is on PATH now (shipped or just installed); silence its update-notifier
	// box so it doesn't clutter the agent session.
	disableNpmUpdateNotifier(ctx)

	// 4. Put npm's global bin on PATH so an npm-installed agent binary resolves after ucode installs it.
	if prefix := npmGlobalPrefix(ctx); prefix != "" {
		prependPath(filepath.Join(prefix, "bin"))
	}

	// 5. Record the resolved PATH so later launches take the fast path.
	if err := os.MkdirAll(filepath.Dir(readyFile), 0o755); err != nil {
		return fmt.Errorf("failed to create %s: %w", filepath.Dir(readyFile), err)
	}
	if err := os.WriteFile(readyFile, []byte(os.Getenv("PATH")), 0o644); err != nil {
		return fmt.Errorf("failed to record setup completion: %w", err)
	}

	return launchUcodeAgent(agent, workspace, agentArgs)
}

// launchUcodeAgent launches the ucode-configured agent, replacing this process.
func launchUcodeAgent(agent agentSpec, workspace string, agentArgs []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to resolve home directory: %w", err)
	}
	ucodePath, err := exec.LookPath("ucode")
	if err != nil {
		return fmt.Errorf("ucode not found on PATH after setup: %w", err)
	}
	contextArgs, err := injectAgentContext(home, agent)
	if err != nil {
		return err
	}
	argv := []string{"ucode", agent.name}
	if workspace != "" {
		// Every supported agent accepts --workspace (the gate for supportedAgents).
		argv = append(argv, "--workspace", workspace)
	}
	argv = append(argv, contextArgs...)
	argv = append(argv, agentArgs...)
	// Pass the session token as DATABRICKS_BEARER so stock ucode authenticates headlessly.
	if os.Getenv("DATABRICKS_BEARER") == "" {
		if token := os.Getenv("DATABRICKS_TOKEN"); token != "" {
			_ = os.Setenv("DATABRICKS_BEARER", token)
		}
	}
	return execProcess(ucodePath, argv, os.Environ())
}

// injectAgentContext puts the Databricks context where the agent reads it, returning any extra argv.
func injectAgentContext(home string, agent agentSpec) ([]string, error) {
	context := agentSystemContext(os.Getenv(workspaceHomeEnv))
	switch {
	case agent.contextFlag != "":
		f := filepath.Join(home, contextFile)
		if err := os.WriteFile(f, []byte(context), 0o644); err != nil {
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
		if err := os.WriteFile(f, []byte(context), 0o644); err != nil {
			return nil, fmt.Errorf("failed to write %s: %w", f, err)
		}
		return nil, nil
	default:
		return nil, nil
	}
}

// ensureNode downloads the latest Krypton LTS Node into deps/node once, returning
// its bin. Extraction output is written to out.
func ensureNode(ctx context.Context, home string, out io.Writer) (string, error) {
	nodeDir := filepath.Join(home, depsDir, "node")
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
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create %s: %w", nodeDir, err)
	}
	// Download to a temp file, verifying its SHA256 from SHASUMS256.txt before use.
	tarball, err := downloadVerified(ctx, base+"/"+tarName, wantSum)
	if err != nil {
		return "", err
	}
	defer os.Remove(tarball)
	// Node's .tar.xz is the smallest download; extract it with the system tar.
	if err := runCommand(ctx, out, "tar", "-xJf", tarball, "--strip-components=1", "-C", nodeDir); err != nil {
		return "", fmt.Errorf("failed to extract Node.js: %w", err)
	}
	return nodeBin, nil
}

// downloadVerified fetches url to a temp file, failing unless its SHA256 matches
// wantSum. The caller is responsible for removing the returned file.
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

// latestUcodeRef returns the tag of ucode's latest GitHub release.
func latestUcodeRef(ctx context.Context, apiURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest ucode release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch latest ucode release: HTTP %d", resp.StatusCode)
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return "", fmt.Errorf("failed to parse latest ucode release: %w", err)
	}
	if tag := strings.TrimSpace(payload.TagName); tag != "" {
		return tag, nil
	}
	return "", errors.New("latest ucode release has empty tag_name")
}

// nodeDownloadArch maps a Go arch to Node's release arch token, or "" if unsupported.
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

// latestNodeTarball returns the linux tarball name and its SHA256 for arch from
// the SHASUMS256.txt listing (lines of "<sha256>  <filename>").
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
	for _, line := range strings.Split(string(body), "\n") {
		if m := re.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			return m[2], m[1], nil
		}
	}
	return "", "", fmt.Errorf("no linux-%s Node tarball found in %s", arch, shasumsURL)
}

// disableNpmUpdateNotifier turns off npm's "new version available" box so it
// doesn't clutter the agent session. Best-effort: it's cosmetic, so a failure
// (e.g. npm not runnable) is logged and ignored rather than blocking the launch.
func disableNpmUpdateNotifier(ctx context.Context) {
	if err := exec.CommandContext(ctx, "npm", "config", "set", "update-notifier", "false").Run(); err != nil {
		log.Debugf(ctx, "failed to disable npm update-notifier: %v", err)
	}
}

// npmGlobalPrefix returns `npm prefix -g`, or "" if npm isn't runnable.
func npmGlobalPrefix(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "npm", "prefix", "-g").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runCommand runs name with the given args, writing combined stdout+stderr to out
// (captured so it surfaces only on failure) and inheriting the process env. Stdin
// is left closed: the shim's install steps are non-interactive.
func runCommand(ctx context.Context, out io.Writer, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// runShell runs a shell snippet — for the curl|sh / curl|tar pipelines — writing
// combined stdout+stderr to out.
func runShell(ctx context.Context, out io.Writer, script string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// prependPath puts dir at the front of $PATH, dropping any later duplicate.
func prependPath(dir string) {
	dirs := []string{dir}
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d != dir {
			dirs = append(dirs, d)
		}
	}
	_ = os.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

// removePath drops every occurrence of dir from $PATH.
func removePath(dir string) {
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d != dir {
			dirs = append(dirs, d)
		}
	}
	_ = os.Setenv("PATH", strings.Join(dirs, string(os.PathListSeparator)))
}

// agentSystemContext returns the Databricks session context; wsHome names the working directory.
func agentSystemContext(wsHome string) string {
	cwd := "the user's Databricks workspace home directory"
	if wsHome != "" {
		cwd = wsHome
	}
	return fmt.Sprintf(`You are running inside a "databricks ssh connect" session on the driver node of a
Databricks serverless cluster.
- The "databricks" CLI is installed and already authenticated: DATABRICKS_HOST and
  DATABRICKS_TOKEN are set in the environment, so "databricks ..." commands work with no
  "databricks auth login". The same token governs Unity Catalog and serving-endpoint access.
- This container is ephemeral; only paths under /Workspace persist across sessions. Your
  working directory is %s.
- DATABRICKS_TOKEN is a static session token that may expire during a long session; if
  "databricks" calls start failing with auth errors, the session likely needs reconnecting.
- You can run shell commands and use the "databricks" CLI to explore the workspace
  (clusters, jobs, Unity Catalog, DBFS, etc.).`, cwd)
}
