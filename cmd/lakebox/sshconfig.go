package lakebox

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
)

const (
	// sshConfigAlias is the SSH-config Host that all lakebox sandboxes
	// route through. Shared with the workspace UI's "First time setup?"
	// disclosure, so IDE Remote-SSH deep links resolve identically
	// whether the user set up via the CLI or pasted the UI snippet.
	sshConfigAlias = "lakebox-gw"

	// sshIncludeFileName is the lakebox-managed file referenced by the
	// user's ~/.ssh/config. The file is fully owned: we rewrite it on
	// every `lakebox register`, so manual edits to it will not survive.
	sshIncludeFileName = "databricks-lakebox"

	// sshConfigBeginMarker / sshConfigEndMarker bracket the single line we
	// add to the user's ~/.ssh/config (the Include directive). Markers make
	// the line greppable and removable without re-parsing the file.
	sshConfigBeginMarker = "# >>> databricks lakebox >>>"
	sshConfigEndMarker   = "# <<< databricks lakebox <<<"
)

// sshConfigPaths returns (managedFile, mainConfig) under the user's
// ~/.ssh directory. Side-effect free; safe to call before deciding
// whether to actually write anything.
func sshConfigPaths(ctx context.Context) (string, string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", "", err
	}
	sshDir := filepath.Join(home, ".ssh")
	return filepath.Join(sshDir, sshIncludeFileName), filepath.Join(sshDir, "config"), nil
}

// sshConfigAlreadyManaged reports whether ~/.ssh/config already
// contains the lakebox-managed Include block. Used by `register` to
// decide whether to prompt the user for consent (first time) or
// silently refresh the managed file (already opted in).
func sshConfigAlreadyManaged(ctx context.Context) (bool, error) {
	_, mainPath, err := sshConfigPaths(ctx)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(mainPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading %s: %w", mainPath, err)
	}
	return hasOurMarkedBlock(string(data)), nil
}

// writeSSHConfig writes the lakebox-managed SSH config block to a
// managed file and, if not already present, adds an Include directive
// to the user's ~/.ssh/config pointing at that file.
//
// Returns (managedFilePath, mainConfigPath, error).
func writeSSHConfig(ctx context.Context, keyPath, gatewayHost, gatewayPort string) (string, string, error) {
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

	block := buildSSHConfigBlock(keyPath, gatewayHost, gatewayPort)
	if err := writeManagedConfig(managedPath, block); err != nil {
		return managedPath, mainPath, err
	}
	if err := ensureMainIncludesManaged(mainPath, managedPath); err != nil {
		return managedPath, mainPath, err
	}
	return managedPath, mainPath, nil
}

// buildSSHConfigBlock renders the Host stanza we write to the
// lakebox-managed include file. The -o flags here mirror buildSSHArgs
// in ssh.go so connections that resolve through this alias (IDE
// Remote-SSH, raw `ssh <id>@lakebox-gw`) behave identically to
// `databricks lakebox ssh`.
//
// No User directive is set, so the per-sandbox identifier travels in
// the destination (`ssh <sandbox-id>@lakebox-gw`); a single alias
// serves every sandbox on this profile's workspace.
func buildSSHConfigBlock(keyPath, gatewayHost, gatewayPort string) string {
	return fmt.Sprintf(`# Managed by `+"`databricks lakebox register`"+`.
# Manual edits will be overwritten on the next run.
Host %s
    HostName %s
    Port %s
    IdentityFile %s
    IdentitiesOnly yes
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
    LogLevel ERROR
`, sshConfigAlias, gatewayHost, gatewayPort, keyPath)
}

// writeManagedConfig writes content to path with 0600 perms, atomically
// via tmp + rename so a crash mid-write can't leave a half-written
// file. Skips the write entirely when the existing content already
// matches, so repeated `lakebox register` runs don't churn the file's
// mtime.
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
	case os.IsNotExist(err):
		existing = nil
	default:
		return fmt.Errorf("reading %s: %w", mainPath, err)
	}

	// Prepend our block. SSH processes the file top-down and uses the
	// first value seen for each option; placing our Include first lets
	// `lakebox-gw` always resolve to the managed values even if the
	// user has a wildcard `Host *` block later.
	var buf bytes.Buffer
	buf.WriteString(managedBlock)
	if len(existing) > 0 {
		// Ensure visual separation between our block and the user's
		// content. If the existing file already starts with a blank
		// line, don't add another.
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
// a lakebox-managed Include block (delimited by our markers).
func hasOurMarkedBlock(text string) bool {
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == sshConfigBeginMarker {
			return true
		}
	}
	return false
}

// maybeWriteSSHConfig writes the lakebox-managed SSH config, prompting
// for consent the first time on this machine. Re-runs are silent: the
// Include line in ~/.ssh/config signals prior opt-in, and we just
// refresh the managed file's contents (e.g. if the gateway host has
// changed).
//
// In non-interactive contexts (no TTY, no `--yes`-style flag here),
// we skip the write rather than fail — `lakebox ssh` still works via
// argv-explicit flags, only IDE Remote-SSH from the workspace UI
// needs the config alias.
func maybeWriteSSHConfig(ctx context.Context, keyPath, workspaceHost string) error {
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
			cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "skipped SSH config update (non-interactive); re-run `databricks lakebox register` from a terminal to add the `"+sshConfigAlias+"` alias"))
			return nil
		}
		question := fmt.Sprintf(
			"Add a `Host %s` alias to %s? This lets editor Remote-SSH (VS Code / Cursor) and `ssh <id>@%s` connect without further setup.",
			sshConfigAlias, mainPath, sshConfigAlias,
		)
		confirmed, err := cmdio.AskYesOrNo(ctx, question)
		if err != nil {
			return err
		}
		if !confirmed {
			cmdio.LogString(ctx, "  "+cmdio.Faint(ctx, "skipped SSH config update; re-run `databricks lakebox register` to revisit"))
			return nil
		}
	}

	gateway := resolveGatewayHost(workspaceHost)
	managedPath, _, err := writeSSHConfig(ctx, keyPath, gateway, defaultGatewayPort)
	if err != nil {
		return err
	}
	ok(ctx, "Updated SSH config (managed: "+cmdio.Faint(ctx, managedPath)+")")
	return nil
}
