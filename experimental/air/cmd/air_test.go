package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewRegistersAllSubcommands asserts the `air` command wires up every
// expected subcommand, so none is accidentally dropped from New.
func TestNewRegistersAllSubcommands(t *testing.T) {
	registered := make(map[string]bool)
	for _, c := range New().Commands() {
		registered[c.Name()] = true
	}

	want := []string{"run", "get", "list", "logs", "cancel", "register-image"}
	for _, name := range want {
		assert.True(t, registered[name], "subcommand %q is not registered", name)
	}
	assert.Len(t, registered, len(want), "unexpected number of subcommands")
}
