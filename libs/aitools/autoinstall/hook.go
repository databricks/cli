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
