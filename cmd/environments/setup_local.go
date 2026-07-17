package environments

import "github.com/spf13/cobra"

// Commands returns the hand-written subcommands to add to the auto-generated
// "environments" command group. They are attached via the group's cmdOverrides
// hook (see cmd/workspace/environments/overrides.go), mirroring how cmd/apps
// extends the generated apps group.
//
// P0 exposes a single verb, "setup-local", which provisions a local Python
// environment matched to a Databricks compute target. It is Python-only and
// takes no language selector (spec §naming); a language axis (setup-local
// python / scala) would be additive if more languages are ever supported.
func Commands() []*cobra.Command {
	return []*cobra.Command{
		newSetupLocalCommand(),
	}
}
