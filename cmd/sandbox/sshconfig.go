package sandbox

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
)

const (
	// sshIncludeFileName is the sandbox-managed file referenced by the
	// user's ~/.ssh/config. The file is fully owned: we rewrite it on
	// every `sandbox register`, so manual edits to it will not survive.
	sshIncludeFileName = "databricks-sandbox"

	// sshConfigBeginMarker / sshConfigEndMarker bracket the single line we
	// add to the user's ~/.ssh/config (the Include directive). Markers make
	// the line greppable and removable without re-parsing the file.
	sshConfigBeginMarker = "# >>> databricks sandbox >>>"
	sshConfigEndMarker   = "# <<< databricks sandbox <<<"
)

// sshConfigPaths returns (managedFile, mainConfig) under the user's
// ~/.ssh directory.
func sshConfigPaths(ctx context.Context) (string, string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", "", err
	}
	sshDir := filepath.Join(home, ".ssh")
	return filepath.Join(sshDir, sshIncludeFileName), filepath.Join(sshDir, "config"), nil
}

// sshConfigAlreadyManaged reports whether ~/.ssh/config already
// contains the sandbox-managed Include block.
func sshConfigAlreadyManaged(ctx context.Context) (bool, error) {
	_, mainPath, err := sshConfigPaths(ctx)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(mainPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", mainPath, err)
	}
	return hasOurMarkedBlock(string(data)), nil
}

// writeSSHConfig writes the sandbox-managed SSH config block to a
// managed file and, if not already present, adds an Include directive
// to the user's ~/.ssh/config pointing at that file.
func writeSSHConfig(ctx context.Context, keyPath string, gatewayHosts []string, gatewayPort string) (string, string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", "", err
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return "", "", fmt.Errorf("creating %s: %w", sshDir, err)
	}

	managedPath := filepath.Join(sshDir, sshIncludeFileName)
	mainPath := filepath.Join(sshDir, "config")

	block := buildSSHConfigBlock(keyPath, gatewayHosts, gatewayPort)
	if err := writeManagedConfig(managedPath, block); err != nil {
		return managedPath, mainPath, err
	}
	if err := ensureMainIncludesManaged(mainPath, managedPath); err != nil {
		return managedPath, mainPath, err
	}
	return managedPath, mainPath, nil
}

// buildSSHConfigBlock renders one Host stanza per gateway. Mirrors the
// snippet the workspace UI's "First time setup?" disclosure recommends
// — the Host key is the literal gateway hostname (so editor Remote-SSH
// deep links like `ssh-remote+<id>@<gateway>` resolve directly), and
// only the two directives that meaningfully differ from SSH defaults
// are set:
//
//   - Port (the gateway listens on 2222, not 22)
//   - IdentityFile + IdentitiesOnly (pin our key so ssh doesn't offer
//     every key in ~/.ssh/ and trip the gateway's rejection cascade)
//
// Notably we do NOT set StrictHostKeyChecking, UserKnownHostsFile, or
// LogLevel — those defaults work fine for IDE / raw-ssh use and
// matching the user's expectations beats the cosmetic suppression the
// CLI's own `sandbox ssh` does via argv. No User directive either —
// the per-sandbox identifier travels in the destination
// (`ssh <sandbox-id>@<gateway>`).
//
// One stanza per gateway means a user with profiles in multiple
// regions doesn't lose IDE Remote-SSH for the earlier workspaces when
// they `register` against a new one. Callers pass `gatewayHosts` from
// state.allGatewayHosts.
func buildSSHConfigBlock(keyPath string, gatewayHosts []string, gatewayPort string) string {
	var b strings.Builder
	b.WriteString("# Managed by `databricks sandbox register`. Manual edits will be overwritten.\n")
	for _, gw := range gatewayHosts {
		fmt.Fprintf(&b, "Host %s\n    Port %s\n    IdentityFile %s\n    IdentitiesOnly yes\n", gw, gatewayPort, keyPath)
	}
	return b.String()
}

// writeManagedConfig writes content to path atomically with 0600
// perms. No-op when the file already matches, to avoid churning mtime.
func writeManagedConfig(path, content string) error {
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, []byte(content)) {
		return nil
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s to %s: %w", tmp, path, err)
	}
	return nil
}

// ensureMainIncludesManaged makes sure ~/.ssh/config begins with an
// `Include <managedPath>` directive bracketed by our begin/end markers.
// If our block is already present, the file is left alone; if absent,
// we prepend the block so it takes precedence over any later Host
// blocks the user has defined (SSH applies the first match per option).
func ensureMainIncludesManaged(mainPath, managedPath string) error {
	managedBlock := fmt.Sprintf("%s\nInclude %s\n%s\n", sshConfigBeginMarker, managedPath, sshConfigEndMarker)

	existing, err := os.ReadFile(mainPath)
	switch {
	case err == nil:
		if hasOurMarkedBlock(string(existing)) {
			return nil
		}
	case errors.Is(err, fs.ErrNotExist):
		existing = nil
	default:
		return fmt.Errorf("reading %s: %w", mainPath, err)
	}

	// Prepend so our Include wins SSH's first-match-per-option
	// semantics over any wildcard `Host *` block later in the file.
	var buf bytes.Buffer
	buf.WriteString(managedBlock)
	if len(existing) > 0 {
		// Avoid double blank lines if the file already starts with one.
		if !strings.HasPrefix(string(existing), "\n") {
			buf.WriteString("\n")
		}
		buf.Write(existing)
	}

	tmp := mainPath + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, mainPath); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("renaming %s to %s: %w", tmp, mainPath, err)
	}
	return nil
}

// hasOurMarkedBlock reports whether the given config text already has
// a sandbox-managed Include block (delimited by our markers).
func hasOurMarkedBlock(text string) bool {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == sshConfigBeginMarker {
			return true
		}
	}
	return false
}

// maybeWriteSSHConfig writes the sandbox-managed SSH config, prompting
// for consent the first time on this machine. Re-runs silently refresh
// the managed file. Non-interactive contexts skip the write entirely.
//
// Reads the gateway-host set from state (populated by every API call
// that touches a Sandbox or SshKey response, including the register
// call the caller just made), so users with multiple workspaces in
// different regions accumulate one Host stanza per unique gateway
// instead of losing the earlier ones.
func maybeWriteSSHConfig(ctx context.Context, keyPath string) error {
	gateways := allGatewayHosts(ctx)
	if len(gateways) == 0 {
		// No gateway has been cached yet — register's API call must
		// not have stamped one, or state is empty. Skip rather than
		// guess; the next API call that returns a Sandbox/SshKey
		// will populate the cache and a future register will write.
		cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "skipped SSH config update — no sandbox gateway is known yet"))
		return nil
	}

	already, err := sshConfigAlreadyManaged(ctx)
	if err != nil {
		return err
	}
	if !already {
		_, mainPath, err := sshConfigPaths(ctx)
		if err != nil {
			return err
		}
		if !cmdio.IsPromptSupported(ctx) {
			cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "skipped SSH config update (non-interactive); re-run `databricks sandbox register` from a terminal to add the sandbox gateway block(s)"))
			return nil
		}
		question := fmt.Sprintf(
			"Add a sandbox `Host` block to %s? This lets editor Remote-SSH (VS Code / Cursor) and `ssh <id>@<gateway>` connect without further setup.",
			mainPath,
		)
		confirmed, err := cmdio.AskYesOrNo(ctx, question)
		if err != nil {
			return err
		}
		if !confirmed {
			cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "skipped SSH config update; re-run `databricks sandbox register` to revisit"))
			return nil
		}
	}

	managedPath, _, err := writeSSHConfig(ctx, keyPath, gateways, defaultGatewayPort)
	if err != nil {
		return err
	}
	ok(ctx, "Updated SSH config (managed: "+cmdio.Faint(ctx, managedPath)+")")
	return nil
}
