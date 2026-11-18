package autoinstall

import "github.com/spf13/cobra"

// Run, when set, is called by the root command after a command succeeded. It
// runs auto-install, which needs the install and update code in cmd/aitools.
// cmd/root cannot import that package (cmd/aitools imports cmd/root), so
// main.go sets Run. Runs that don't start in main.go, such as in-process
// tests, leave it nil and never auto-install.
var Run func(cmd *cobra.Command)
