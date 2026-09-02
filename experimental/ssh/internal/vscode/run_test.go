package vscode

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckIDECommand_Missing(t *testing.T) {
	// Override PATH to ensure commands are not found
	t.Setenv("PATH", t.TempDir())

	tests := []struct {
		name       string
		ide        string
		wantErrMsg string
	}{
		{
			name:       "missing vscode command",
			ide:        VSCodeOption,
			wantErrMsg: `"code" command not found on PATH`,
		},
		{
			name:       "missing cursor command",
			ide:        CursorOption,
			wantErrMsg: `"cursor" command not found on PATH`,
		},
		{
			name:       "vscode error contains install instructions",
			ide:        VSCodeOption,
			wantErrMsg: "https://code.visualstudio.com/",
		},
		{
			name:       "cursor error contains install instructions",
			ide:        CursorOption,
			wantErrMsg: "https://cursor.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckIDECommand(tt.ide)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrMsg)
		})
	}
}

func TestCheckIDECommand_Found(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)

	tests := []struct {
		name    string
		ide     string
		command string
	}{
		{
			name:    "vscode command found",
			ide:     VSCodeOption,
			command: "code",
		},
		{
			name:    "cursor command found",
			ide:     CursorOption,
			command: "cursor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake executable in the temp directory.
			// On Windows, exec.LookPath requires a known extension (e.g. .exe).
			command := tt.command
			if runtime.GOOS == "windows" {
				command += ".exe"
			}
			fakePath := filepath.Join(tmpDir, command)
			err := os.WriteFile(fakePath, []byte("#!/bin/sh\n"), 0o755)
			require.NoError(t, err)

			err = CheckIDECommand(tt.ide)
			assert.NoError(t, err)
		})
	}
}

func TestParseExtensionVersion(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		extensionID string
		wantVersion string
		wantFound   bool
		minVersion  string
		wantAtLeast bool
	}{
		{
			name:        "found and above minimum",
			output:      "ms-python.python@2024.1.1\nms-vscode-remote.remote-ssh@0.123.0\n",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "0.123.0",
			wantFound:   true,
			minVersion:  "0.120.0",
			wantAtLeast: true,
		},
		{
			name:        "found but below minimum",
			output:      "ms-vscode-remote.remote-ssh@0.100.0\n",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "0.100.0",
			wantFound:   true,
			minVersion:  "0.120.0",
			wantAtLeast: false,
		},
		{
			name:        "not found",
			output:      "ms-python.python@2024.1.1\n",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "",
			wantFound:   false,
		},
		{
			name:        "empty output",
			output:      "",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "",
			wantFound:   false,
		},
		{
			name:        "multiple extensions",
			output:      "ext.a@1.0.0\next.b@2.0.0\next.c@3.0.0\n",
			extensionID: "ext.b",
			wantVersion: "2.0.0",
			wantFound:   true,
			minVersion:  "1.0.0",
			wantAtLeast: true,
		},
		{
			name:        "prerelease is less than release",
			output:      "ms-vscode-remote.remote-ssh@0.120.0-beta.1\n",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "0.120.0-beta.1",
			wantFound:   true,
			minVersion:  "0.120.0",
			wantAtLeast: false,
		},
		{
			name:        "line with whitespace",
			output:      "  ms-vscode-remote.remote-ssh@0.123.0  \n",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "0.123.0",
			wantFound:   true,
			minVersion:  "0.120.0",
			wantAtLeast: true,
		},
		{
			name:        "windows CRLF line endings",
			output:      "ms-python.python@2024.1.1\r\nms-vscode-remote.remote-ssh@0.123.0\r\n",
			extensionID: "ms-vscode-remote.remote-ssh",
			wantVersion: "0.123.0",
			wantFound:   true,
			minVersion:  "0.120.0",
			wantAtLeast: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, found := parseExtensionVersion(tt.output, tt.extensionID)
			assert.Equal(t, tt.wantFound, found)
			assert.Equal(t, tt.wantVersion, version)
			if found {
				assert.Equal(t, tt.wantAtLeast, isExtensionVersionAtLeast(version, tt.minVersion))
			}
		})
	}
}

func TestIsExtensionVersionAtLeast(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		minVersion string
		want       bool
	}{
		{name: "above minimum", version: "0.123.0", minVersion: "0.120.0", want: true},
		{name: "exact minimum", version: "0.120.0", minVersion: "0.120.0", want: true},
		{name: "below minimum", version: "0.100.0", minVersion: "0.120.0", want: false},
		{name: "major version ahead", version: "1.0.0", minVersion: "0.120.0", want: true},
		{name: "prerelease below release", version: "0.120.0-beta.1", minVersion: "0.120.0", want: false},
		{name: "prerelease above prior release", version: "0.121.0-beta.1", minVersion: "0.120.0", want: true},
		{name: "two-component version is valid", version: "1.0", minVersion: "0.120.0", want: true},
		{name: "empty version", version: "", minVersion: "0.120.0", want: false},
		{name: "garbage version", version: "abc", minVersion: "0.120.0", want: false},
		{name: "four-component version is invalid", version: "0.120.0.1", minVersion: "0.120.0", want: false},
		{name: "cursor exact minimum", version: "1.0.32", minVersion: "1.0.32", want: true},
		{name: "cursor above minimum", version: "1.1.0", minVersion: "1.0.32", want: true},
		{name: "cursor below minimum", version: "1.0.31", minVersion: "1.0.32", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isExtensionVersionAtLeast(tt.version, tt.minVersion))
		})
	}
}

// createFakeIDEExecutable creates a fake IDE command that outputs the given text
// when called with --list-extensions --show-versions.
func createFakeIDEExecutable(t *testing.T, dir, command, output string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		// Write output to a temp file and use "type" to print it, avoiding escaping issues.
		payloadPath := filepath.Join(dir, command+"-payload.txt")
		err := os.WriteFile(payloadPath, []byte(output), 0o644)
		require.NoError(t, err)
		script := fmt.Sprintf("@echo off\ntype \"%s\"\n", payloadPath)
		err = os.WriteFile(filepath.Join(dir, command+".cmd"), []byte(script), 0o755)
		require.NoError(t, err)
	} else {
		// Use printf (a shell builtin) instead of cat to avoid PATH issues in tests.
		script := fmt.Sprintf("#!/bin/sh\nprintf '%%s' '%s'\n", output)
		err := os.WriteFile(filepath.Join(dir, command), []byte(script), 0o755)
		require.NoError(t, err)
	}
}

// createFailingIDEExecutable writes a fake IDE command that exits non-zero for every
// invocation, so "--list-extensions" fails and the check never learns what is installed.
func createFailingIDEExecutable(t *testing.T, dir, command string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		err := os.WriteFile(filepath.Join(dir, command+".cmd"), []byte("@echo off\nexit /b 4\n"), 0o755)
		require.NoError(t, err)
	} else {
		err := os.WriteFile(filepath.Join(dir, command), []byte("#!/bin/sh\nexit 4\n"), 0o755)
		require.NoError(t, err)
	}
}

// createIDEExecutableFailingInstall writes a fake IDE command that lists extensions
// successfully but rejects "--install-extension", as a marketplace or policy block would.
func createIDEExecutableFailingInstall(t *testing.T, dir, command, output string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		payloadPath := filepath.Join(dir, command+"-payload.txt")
		err := os.WriteFile(payloadPath, []byte(output), 0o644)
		require.NoError(t, err)
		script := fmt.Sprintf("@echo off\nif \"%%1\"==\"--install-extension\" exit /b 3\ntype \"%s\"\n", payloadPath)
		err = os.WriteFile(filepath.Join(dir, command+".cmd"), []byte(script), 0o755)
		require.NoError(t, err)
	} else {
		script := fmt.Sprintf("#!/bin/sh\nfor a in \"$@\"; do\n  [ \"$a\" = \"--install-extension\" ] && exit 3\ndone\nprintf '%%s' '%s'\n", output)
		err := os.WriteFile(filepath.Join(dir, command), []byte(script), 0o755)
		require.NoError(t, err)
	}
}

func TestCheckIDESSHExtension_UpToDate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	extensionOutput := "ms-python.python@2024.1.1\nms-vscode-remote.remote-ssh@0.123.0\n"
	createFakeIDEExecutable(t, tmpDir, "code", extensionOutput)

	err := CheckIDESSHExtension(ctx, VSCodeOption, false)
	assert.NoError(t, err)
}

func TestCheckIDESSHExtension_ExactMinVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	extensionOutput := "ms-vscode-remote.remote-ssh@0.120.0\n"
	createFakeIDEExecutable(t, tmpDir, "code", extensionOutput)

	err := CheckIDESSHExtension(ctx, VSCodeOption, false)
	assert.NoError(t, err)
}

func TestCheckIDESSHExtension_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	extensionOutput := "ms-python.python@2024.1.1\n"
	createFakeIDEExecutable(t, tmpDir, "code", extensionOutput)

	err := CheckIDESSHExtension(ctx, VSCodeOption, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"Remote - SSH"`)
	assert.Contains(t, err.Error(), "not installed")
	// The test context is not a TTY, so consent cannot be asked for.
	assert.ErrorIs(t, err, ErrSSHExtensionInstallUnavailable)
}

func TestCheckIDESSHExtension_Outdated(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	extensionOutput := "ms-vscode-remote.remote-ssh@0.100.0\n"
	createFakeIDEExecutable(t, tmpDir, "code", extensionOutput)

	err := CheckIDESSHExtension(ctx, VSCodeOption, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "0.100.0")
	assert.Contains(t, err.Error(), ">= 0.120.0")
	assert.ErrorIs(t, err, ErrSSHExtensionInstallUnavailable)
}

func TestCheckIDESSHExtension_Cursor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	extensionOutput := "anysphere.remote-ssh@1.0.32\n"
	createFakeIDEExecutable(t, tmpDir, "cursor", extensionOutput)

	err := CheckIDESSHExtension(ctx, CursorOption, false)
	assert.NoError(t, err)
}

func TestCheckIDESSHExtension_AutoApproveMissing_Installs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	// Fake `code` returns no extensions for --list-extensions, but succeeds for --install-extension.
	createFakeIDEExecutable(t, tmpDir, "code", "")

	err := CheckIDESSHExtension(ctx, VSCodeOption, true)
	assert.NoError(t, err)
}

func TestCheckIDESSHExtension_NoPrompt_WithoutAutoApprove_Errors(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	extensionOutput := "ms-python.python@2024.1.1\n"
	createFakeIDEExecutable(t, tmpDir, "code", extensionOutput)

	err := CheckIDESSHExtension(ctx, VSCodeOption, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--install-extension")
	assert.ErrorIs(t, err, ErrSSHExtensionInstallUnavailable)
}

// A command that is on PATH but whose --list-extensions fails is reported separately from a
// missing extension: nothing was learned about what is installed, so it is not an install
// problem. CheckIDECommand passes here, since the command does resolve.
func TestCheckIDESSHExtension_ListFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	createFailingIDEExecutable(t, tmpDir, "code")

	err := CheckIDESSHExtension(ctx, VSCodeOption, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSSHExtensionListFailed)
	assert.NotErrorIs(t, err, ErrSSHExtensionInstallFailed)
}

// With --auto-approve there is no prompt, so a missing extension goes straight to an install.
// A rejected install is the one outcome the IDE button can produce on this path.
func TestCheckIDESSHExtension_AutoApprove_InstallFails(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("PATH", tmpDir)
	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())

	createIDEExecutableFailingInstall(t, tmpDir, "code", "ms-python.python@2024.1.1\n")

	err := CheckIDESSHExtension(ctx, VSCodeOption, true)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSSHExtensionInstallFailed)
	assert.NotErrorIs(t, err, ErrSSHExtensionInstallUnavailable)
}
