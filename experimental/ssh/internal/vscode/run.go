package vscode

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/cmdio"
)

const (
	// Options as they can be set via --ide flag
	VSCodeOption = "vscode"
	CursorOption = "cursor"
	// CLI commands to launch IDEs
	vscodeCommand = "code"
	cursorCommand = "cursor"
	// Human-readable names to show in the output
	vscodeName = "VS Code"
	cursorName = "Cursor"
)

// CheckIDECommand verifies the IDE CLI command is available on PATH.
func CheckIDECommand(ide string) error {
	ideCommand, ideName, installURL := ideInfo(ide)

	if _, err := exec.LookPath(ideCommand); err != nil {
		return fmt.Errorf(
			"%q command not found on PATH. To fix this:\n"+
				"1. Install %s from %s\n"+
				"2. Open the Command Palette (Cmd+Shift+P / Ctrl+Shift+P) and run \"Shell Command: Install '%s' command\"\n"+
				"3. Restart your terminal",
			ideCommand, ideName, installURL, ideCommand,
		)
	}
	return nil
}

// LaunchIDE launches the IDE with a remote SSH connection.
func LaunchIDE(ctx context.Context, ideOption, connectionName, userName, databricksUserName string) error {
	ideCommand, _, _ := ideInfo(ideOption)

	// Construct the remote SSH URI
	// Format: ssh-remote+<server_user_name>@<connection_name> /Workspace/Users/<databricks_user_name>/
	remoteURI := fmt.Sprintf("ssh-remote+%s@%s", userName, connectionName)
	remotePath := fmt.Sprintf("/Workspace/Users/%s/", databricksUserName)

	cmdio.LogString(ctx, fmt.Sprintf("Launching %s with remote URI: %s and path: %s", ideOption, remoteURI, remotePath))

	ideCmd := exec.CommandContext(ctx, ideCommand, "--remote", remoteURI, remotePath)
	ideCmd.Stdout = os.Stdout
	ideCmd.Stderr = os.Stderr

	return ideCmd.Run()
}

func ideName(ideOption string) string {
	_, name, _ := ideInfo(ideOption)
	return name
}

func ideInfo(ideOption string) (command, name, installURL string) {
	if ideOption == CursorOption {
		return cursorCommand, cursorName, "https://cursor.com/"
	}
	return vscodeCommand, vscodeName, "https://code.visualstudio.com/"
}
