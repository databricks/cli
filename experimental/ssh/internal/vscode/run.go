package vscode

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

// Options as they can be set via --ide flag.
const (
	VSCodeOption = "vscode"
	CursorOption = "cursor"
)

type ideDescriptor struct {
	Option                 string
	Command                string
	Name                   string
	InstallURL             string
	AppName                string
	SSHExtensionID         string
	SSHExtensionName       string
	MinSSHExtensionVersion string
	// DefaultExtensions are seeded into the remote IDE's
	// "remote.SSH.defaultExtensions" setting so they auto-install on the remote.
	DefaultExtensions []string
	// IncompatibleExtensions are removed from "remote.SSH.defaultExtensions" if
	// present. Used to heal settings written by older CLI builds that seeded
	// extensions which hang this IDE's remote auto-install.
	IncompatibleExtensions []string
	// LaunchArgs are extra flags passed to the IDE command before "--remote"
	// when opening the remote window.
	LaunchArgs []string
}

var vsCodeIDE = ideDescriptor{
	Option:           VSCodeOption,
	Command:          "code",
	Name:             "VS Code",
	InstallURL:       "https://code.visualstudio.com/",
	AppName:          "Code",
	SSHExtensionID:   "ms-vscode-remote.remote-ssh",
	SSHExtensionName: "Remote - SSH",
	// Earlier versions might work too, 0.120.0 is a safe not-too-old pick
	MinSSHExtensionVersion: "0.120.0",
	DefaultExtensions:      []string{pythonExtension, jupyterExtension, databricksExtension},
}

var cursorIDE = ideDescriptor{
	Option:           CursorOption,
	Command:          "cursor",
	Name:             "Cursor",
	InstallURL:       "https://cursor.com/",
	AppName:          "Cursor",
	SSHExtensionID:   "anysphere.remote-ssh",
	SSHExtensionName: "Remote - SSH",
	// Earlier versions don't support remote.SSH.serverPickPortsFromRange option
	MinSSHExtensionVersion: "1.0.32",
	// Cursor's marketplace silently remaps ms-python.python -> anysphere.python
	// (and Pylance -> anysphere.cursorpyright), which forms a circular dependency
	// and hangs the remote auto-install indefinitely. Cursor ships its own Python
	// stack anyway, so we only seed the Databricks extension here and let the user
	// (or that extension) pull in Python/Jupyter as needed. See DECO-27339.
	DefaultExtensions: []string{databricksExtension},
	// Heal settings written by older CLI builds: these are the extensions that
	// trigger the Cursor remap hang, so strip them from defaultExtensions if a
	// prior `ssh connect --ide cursor` left them behind. See DECO-27339.
	IncompatibleExtensions: []string{pythonExtension, jupyterExtension},
	// Cursor 3.x defaults new windows to the "Agents Window", which swallows the
	// "--remote" request and never opens the remote editor. --classic forces a
	// classic editor window so the remote workspace actually opens. See DECO-27339.
	LaunchArgs: []string{"--classic"},
}

func getIDE(option string) ideDescriptor {
	if option == CursorOption {
		return cursorIDE
	}
	return vsCodeIDE
}

// CheckIDECommand verifies the IDE CLI command is available on PATH.
func CheckIDECommand(option string) error {
	ide := getIDE(option)

	if _, err := exec.LookPath(ide.Command); err != nil {
		return fmt.Errorf(
			"%q command not found on PATH. To fix this:\n"+
				"1. Install %s from %s\n"+
				"2. Open the Command Palette (Cmd+Shift+P / Ctrl+Shift+P) and run \"Shell Command: Install '%s' command\"\n"+
				"3. Restart your terminal",
			ide.Command, ide.Name, ide.InstallURL, ide.Command,
		)
	}
	return nil
}

// parseExtensionVersion finds the version of the given extension in the output
// of "<command> --list-extensions --show-versions" (one "name@version" per line).
func parseExtensionVersion(output, extensionID string) (string, bool) {
	for line := range strings.SplitSeq(output, "\n") {
		name, version, ok := strings.Cut(strings.TrimSpace(line), "@")
		if ok && name == extensionID {
			return version, true
		}
	}
	return "", false
}

func isExtensionVersionAtLeast(version, minVersion string) bool {
	v := "v" + version
	return semver.IsValid(v) && semver.Compare(v, "v"+minVersion) >= 0
}

// CheckIDESSHExtension verifies that the required Remote SSH extension is installed
// with a compatible version, and offers to install/update it if not.
// When autoApprove is true, the extension is installed without asking.
func CheckIDESSHExtension(ctx context.Context, option string, autoApprove bool) error {
	ide := getIDE(option)

	out, err := exec.CommandContext(ctx, ide.Command, "--list-extensions", "--show-versions").Output()
	if err != nil {
		return fmt.Errorf("failed to list %s extensions: %w", ide.Name, err)
	}

	version, found := parseExtensionVersion(string(out), ide.SSHExtensionID)
	if found && isExtensionVersionAtLeast(version, ide.MinSSHExtensionVersion) {
		return nil
	}

	var msg string
	if !found {
		msg = fmt.Sprintf("Required extension %q is not installed in %s.", ide.SSHExtensionName, ide.Name)
	} else {
		msg = fmt.Sprintf("Extension %q version %s is installed, but version >= %s is required.",
			ide.SSHExtensionName, version, ide.MinSSHExtensionVersion)
	}

	if !autoApprove {
		if !cmdio.IsPromptSupported(ctx) {
			return fmt.Errorf("%s Install it with: %s --install-extension %s, or pass --auto-approve",
				msg, ide.Command, ide.SSHExtensionID)
		}

		shouldInstall, err := cmdio.AskYesOrNo(ctx, msg+" Would you like to install it?")
		if err != nil {
			return fmt.Errorf("failed to prompt user: %w", err)
		}
		if !shouldInstall {
			return fmt.Errorf("%s Install it with: %s --install-extension %s",
				msg, ide.Command, ide.SSHExtensionID)
		}
	} else {
		cmdio.LogString(ctx, msg+" Installing automatically (--auto-approve).")
	}

	cmdio.LogString(ctx, fmt.Sprintf("Installing %q...", ide.SSHExtensionName))
	installCmd := exec.CommandContext(ctx, ide.Command, "--install-extension", ide.SSHExtensionID, "--force")
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr
	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install extension %q: %w", ide.SSHExtensionName, err)
	}
	return nil
}

// LaunchIDE launches the IDE with a remote SSH connection using special "ssh-remote" URI format.
func LaunchIDE(ctx context.Context, ideOption, connectionName, userName, databricksUserName string) error {
	ide := getIDE(ideOption)

	// Construct the remote SSH URI
	// Format: ssh-remote+<server_user_name>@<connection_name> /Workspace/Users/<databricks_user_name>/
	remoteURI := fmt.Sprintf("ssh-remote+%s@%s", userName, connectionName)
	remotePath := fmt.Sprintf("/Workspace/Users/%s/", databricksUserName)

	log.Infof(ctx, "Launching %s with remote URI: %s and path: %s", ideOption, remoteURI, remotePath)

	args := append(append([]string{}, ide.LaunchArgs...), "--remote", remoteURI, remotePath)
	ideCmd := exec.CommandContext(ctx, ide.Command, args...)
	ideCmd.Stdout = os.Stdout
	ideCmd.Stderr = os.Stderr

	return ideCmd.Run()
}
