package generate

import (
	"fmt"
	"maps"
	"slices"

	"github.com/spf13/cobra"
)

// generatorCommands is keyed by the resource key under "resources:" in
// databricks.yml, so the subcommand order does not depend on the order in
// which generator files register.
var generatorCommands = map[string]func() *cobra.Command{}

// registerGenerator adds a "bundle generate" subcommand for the resource type
// with the given key. Call it from the generator's file in func init().
func registerGenerator(resourceKey string, newCommand func() *cobra.Command) {
	if _, ok := generatorCommands[resourceKey]; ok {
		panic(fmt.Sprintf("generator for %q registered twice", resourceKey))
	}
	generatorCommands[resourceKey] = newCommand
}

// NewGenerateCommands returns the "bundle generate" subcommands of all
// registered generators, ordered by resource key.
func NewGenerateCommands() []*cobra.Command {
	var cmds []*cobra.Command
	for _, resourceKey := range slices.Sorted(maps.Keys(generatorCommands)) {
		cmds = append(cmds, generatorCommands[resourceKey]())
	}
	return cmds
}
