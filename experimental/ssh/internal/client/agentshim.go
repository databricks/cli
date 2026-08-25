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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
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

// probeAIGateway fails fast if the workspace's AI Gateway is unreachable (ucode's v2→v3 probe).
func probeAIGateway(ctx context.Context, client *databricks.WorkspaceClient) error {
	host := strings.TrimRight(client.Config.Host, "/")
	// Mirror ucode: try the AI Gateway v2 endpoints API, then fall back to v3.
	var lastStatus int
	for _, path := range []string{
		"/api/ai-gateway/v2/endpoints?page_size=1",
		"/api/2.1/unity-catalog/model-services?page_size=1",
	} {
		status, err := gatewayProbe(ctx, client, host+path)
		if err != nil {
			return err
		}
		if status == http.StatusOK {
			return nil
		}
		lastStatus = status
	}
	return fmt.Errorf("Unity AI Gateway is not enabled on this workspace (%s): probe returned HTTP %d. Enable the AI Gateway (Foundation Model APIs) and try again", host, lastStatus)
}

// gatewayProbe issues an authenticated GET and returns the HTTP status; err is
// non-nil only for build/auth/transport failures, not a non-200 response.
func gatewayProbe(ctx context.Context, client *databricks.WorkspaceClient, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build AI Gateway probe request: %w", err)
	}
	if err := client.Config.Authenticate(req); err != nil {
		return 0, fmt.Errorf("failed to authenticate AI Gateway probe: %w", err)
	}
	resp, err := (&http.Client{Transport: client.Config.HTTPTransport}).Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to reach the Unity AI Gateway: %w", err)
	}
	resp.Body.Close()
	return resp.StatusCode, nil
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

	// 1. uv (installs into ~/.local/bin).
	if _, err := exec.LookPath("uv"); err != nil {
		cmdio.LogString(ctx, "Installing uv...")
		if err := runShell(ctx, "curl -LsSf https://astral.sh/uv/install.sh | sh"); err != nil {
			return fmt.Errorf("failed to install uv: %w", err)
		}
	}

	// 2. ucode (latest stock upstream release).
	if _, err := exec.LookPath("ucode"); err != nil {
		cmdio.LogString(ctx, "Installing ucode...")
		ref, err := latestUcodeRef(ctx, "https://api.github.com/repos/"+ucodeRepo+"/releases/latest")
		if err != nil {
			return err
		}
		if err := runCommand(ctx, "uv", "tool", "install", "git+https://github.com/"+ucodeRepo+"@"+ref); err != nil {
			return fmt.Errorf("failed to install ucode: %w", err)
		}
	}

	// 3. Node/npm. Some agents (e.g. Claude Code) need it and serverless images don't ship it.
	if _, err := exec.LookPath("npm"); err != nil {
		cmdio.LogString(ctx, "Installing npm...")
		nodeBin, err := ensureNode(ctx, home)
		if err != nil {
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

// ensureNode downloads the latest Krypton LTS Node into deps/node once, returning its bin.
func ensureNode(ctx context.Context, home string) (string, error) {
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
	if err := runCommand(ctx, "tar", "-xJf", tarball, "--strip-components=1", "-C", nodeDir); err != nil {
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

// runCommand runs name with the given args, inheriting stdio and the process env.
func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runShell runs a shell snippet — for the curl|sh / curl|tar pipelines.
func runShell(ctx context.Context, script string) error {
	cmd := exec.CommandContext(ctx, "sh", "-c", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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
