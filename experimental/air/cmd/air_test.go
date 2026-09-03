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

// convert-to-dabs has graduated to the stable top-level `air` group; it must live
// there and no longer under experimental.
func TestNewStableRegistersConvertToDabs(t *testing.T) {
	stable := make(map[string]bool)
	for _, c := range NewStable().Commands() {
		stable[c.Name()] = true
	}
	assert.True(t, stable["convert-to-dabs"], "convert-to-dabs is not registered on the stable air group")

	experimental := make(map[string]bool)
	for _, c := range New().Commands() {
		experimental[c.Name()] = true
	}
	assert.False(t, experimental["convert-to-dabs"], "convert-to-dabs must not remain under experimental air")
}
