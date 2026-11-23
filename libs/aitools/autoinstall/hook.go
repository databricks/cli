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

// InstallsFor, when set, reports whether the Run after cmd is due to give
// the agent with this registry name its first Databricks AI tools. The install
// hint, printed before the command runs, is withheld then so the agent is not
// told to install what the same run installs. It reads only local state: the
// gate is looked up after the command, so a run it reports may still end
// without installing, and the hint then waits for the next command. main.go
// sets it with Run.
var InstallsFor func(cmd *cobra.Command, agentName string) bool
