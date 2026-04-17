package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
)

const (
	updateCheckURL     = "https://api.github.com/repos/databricks/cli/releases/latest"
	updateCheckTimeout = 3 * time.Second
)

type execFunc func(ctx context.Context, name string, args ...string) (string, error)

func realExec(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// checkToolchain reports the versions of external tools the CLI commonly shells out to.
// Missing tools are reported inline but do not fail the check, since none are strictly required.
func checkToolchain(ctx context.Context, run execFunc) CheckResult {
	tools := []struct {
		name string
		cmd  string
		arg  string
	}{
		{"git", "git", "--version"},
		{"python", "python3", "--version"},
		{"uv", "uv", "--version"},
		{"terraform", "terraform", "-version"},
	}

	parts := make([]string, 0, len(tools))
	for _, t := range tools {
		out, err := run(ctx, t.cmd, t.arg)
		if err != nil {
			parts = append(parts, t.name+" not found")
			continue
		}
		parts = append(parts, t.name+" "+firstLineVersion(out))
	}
	return info("Toolchain", strings.Join(parts, ", "))
}

// firstLineVersion returns a short version string from a tool's --version output.
// It takes the first non-empty line and strips common program-name prefixes
// (e.g. "git version 2.42.0" -> "2.42.0", "Terraform v1.5.0" -> "v1.5.0").
func firstLineVersion(out string) string {
	line := strings.TrimSpace(strings.SplitN(out, "\n", 2)[0])
	fields := strings.Fields(line)
	for i, f := range fields {
		if looksLikeVersion(f) {
			return strings.Join(fields[i:], " ")
		}
	}
	return line
}

func looksLikeVersion(tok string) bool {
	if len(tok) == 0 {
		return false
	}
	c := tok[0]
	if c >= '0' && c <= '9' {
		return true
	}
	// Accept a leading "v" only when followed by a digit (e.g. v1.5.0).
	if c == 'v' && len(tok) >= 2 && tok[1] >= '0' && tok[1] <= '9' {
		return true
	}
	return false
}

// proxyEnvVars lists the environment variables that influence HTTP/TLS behavior.
// Both upper- and lower-case forms are checked since Go's http.ProxyFromEnvironment
// honors both.
var proxyEnvVars = []string{
	"HTTPS_PROXY", "https_proxy",
	"HTTP_PROXY", "http_proxy",
	"NO_PROXY", "no_proxy",
	"SSL_CERT_FILE",
	"REQUESTS_CA_BUNDLE",
	"CURL_CA_BUNDLE",
}

// checkProxy reports proxy/TLS-related environment variables that affect the CLI's
// network stack. These are a common source of enterprise connectivity issues.
func checkProxy(ctx context.Context) CheckResult {
	var seen []string
	reported := map[string]bool{}
	for _, key := range proxyEnvVars {
		v := env.Get(ctx, key)
		if v == "" {
			continue
		}
		canonical := strings.ToUpper(key)
		if reported[canonical] {
			continue
		}
		reported[canonical] = true
		seen = append(seen, canonical+"="+maskProxyValue(v))
	}
	if len(seen) == 0 {
		return info("Proxy/TLS", "no proxy or TLS overrides configured")
	}
	return info("Proxy/TLS", strings.Join(seen, ", "))
}

// maskProxyValue hides credentials in proxy URLs (user:pass@host).
// net/url would percent-encode "***", so the replacement is done via string rewrite
// on the exact userinfo segment.
func maskProxyValue(v string) string {
	u, err := url.Parse(v)
	if err != nil || u.User == nil {
		return v
	}
	userinfo := u.User.String()
	if _, hasPassword := u.User.Password(); !hasPassword {
		return v
	}
	masked := u.User.Username() + ":***"
	return strings.Replace(v, userinfo, masked, 1)
}

// checkUpdates fetches the latest CLI release and compares it to the current build.
// Development builds are reported as info; older builds produce a warn status.
func checkUpdates(ctx context.Context, client *http.Client, releaseURL string) CheckResult {
	return checkUpdatesWithVersion(ctx, client, releaseURL, build.GetInfo().Version)
}

// checkUpdatesWithVersion is the testable core of checkUpdates, parameterized on the current version.
func checkUpdatesWithVersion(ctx context.Context, client *http.Client, releaseURL, current string) CheckResult {
	if isDevBuild(current) {
		return info("Updates", "development build ("+current+")")
	}

	ctx, cancel := context.WithTimeout(ctx, updateCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return warn("Updates", "cannot check for updates: "+err.Error())
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return warn("Updates", "cannot reach "+releaseURL)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return warn("Updates", "cannot read release response: "+err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return warn("Updates", fmt.Sprintf("release lookup returned HTTP %d", resp.StatusCode))
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return warn("Updates", "cannot parse release response: "+err.Error())
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	if latest == "" {
		return warn("Updates", "empty release tag")
	}
	if latest == current {
		return pass("Updates", "up to date ("+current+")")
	}
	return warn("Updates", fmt.Sprintf("newer version available: %s (current %s)", latest, current))
}

// isDevBuild reports whether the running binary is a local/dev build.
// The build.GetInfo version for dev builds is "0.0.0-dev+<sha>".
func isDevBuild(version string) bool {
	return version == "" || strings.HasPrefix(version, "0.0.0-dev")
}

// checkLogFile surfaces where CLI logs are being written, so users can find them for support.
func checkLogFile(ctx context.Context) CheckResult {
	path := env.Get(ctx, "DATABRICKS_LOG_FILE")
	if path == "" {
		return info("Log File", "not configured (set DATABRICKS_LOG_FILE or pass --log-file to enable)")
	}
	return info("Log File", path)
}
