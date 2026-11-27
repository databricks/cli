package autoinstall

import (
	"os"

	"github.com/spf13/cobra"
)

// Run, when set, is called by the root command after a command succeeded. It
// runs auto-install, which needs the install and update code in cmd/aitools.
// cmd/root cannot import that package (cmd/aitools imports cmd/root), so
// main.go sets Run. Runs that don't start in main.go, such as in-process
// tests, leave it nil and never auto-install.
//
// Run returns the signal that stopped auto-install, if one did. The root
// command ends the CLI by it once it uploaded the telemetry and closed the log
// file.
var Run func(cmd *cobra.Command) os.Signal

// AskAtLogin, when set, is called by `databricks auth login` with its command
// after the login saved the profile. Where the question applies, it asks the
// user once per machine whether to install Databricks AI tools and keep them
// up to date; after a "yes", the Run after the same command installs them.
// Ctrl+C at the question makes that Run return the signal. It needs the
// planning code of cmd/aitools, which cmd/auth must not import, so main.go
// sets it with Run. It never fails the command and never writes to stdout.
var AskAtLogin func(cmd *cobra.Command)

// InstallsFor, when set, reports whether the Run after cmd may give the agent
// with this registry name Databricks AI tools. The install hint, printed before
// the command runs, is withheld then so the agent is not told to install what
// the same run installs. It reads only local state: the gate is looked up
// after the command, Run is skipped when the command fails, and an install can
// fail, so the withheld hint prints when the command ends unless the agent has
// the tools by then. main.go sets it with Run.
var InstallsFor func(cmd *cobra.Command, agentName string) bool
