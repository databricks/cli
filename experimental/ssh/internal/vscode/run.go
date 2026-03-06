package vscode

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/cmdio"
)

// Options as they can be set via --ide flag.
const (
	VSCodeOption = "vscode"
	CursorOption = "cursor"
)

type ideDescriptor struct {
	Option     string
	Command    string
	Name       string
	InstallURL string
	AppName    string
}

var vsCodeIDE = ideDescriptor{
	Option:     VSCodeOption,
	Command:    "code",
	Name:       "VS Code",
	InstallURL: "https://code.visualstudio.com/",
	AppName:    "Code",
}

var cursorIDE = ideDescriptor{
	Option:     CursorOption,
	Command:    "cursor",
	Name:       "Cursor",
	InstallURL: "https://cursor.com/",
	AppName:    "Cursor",
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

// LaunchIDE launches the IDE with a remote SSH connection.
func LaunchIDE(ctx context.Context, ideOption, connectionName, userName, databricksUserName string) error {
	ide := getIDE(ideOption)

	// Construct the remote SSH URI
	// Format: ssh-remote+<server_user_name>@<connection_name> /Workspace/Users/<databricks_user_name>/
	remoteURI := fmt.Sprintf("ssh-remote+%s@%s", userName, connectionName)
	remotePath := fmt.Sprintf("/Workspace/Users/%s/", databricksUserName)

	cmdio.LogString(ctx, fmt.Sprintf("Launching %s with remote URI: %s and path: %s", ideOption, remoteURI, remotePath))

	ideCmd := exec.CommandContext(ctx, ide.Command, "--remote", remoteURI, remotePath)
	ideCmd.Stdout = os.Stdout
	ideCmd.Stderr = os.Stderr

	return ideCmd.Run()
}
