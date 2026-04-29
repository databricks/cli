package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew_RegistersBindAndUnbind(t *testing.T) {
	cmd := New()
	got := map[string]bool{}
	for _, sub := range cmd.Commands() {
		got[sub.Name()] = true
	}
	assert.True(t, got["bind"], "bind subcommand missing")
	assert.True(t, got["unbind"], "unbind subcommand missing")
	assert.True(t, got["migrate"], "migrate subcommand missing")
}

func TestBind_And_Unbind_AutoApproveFlag(t *testing.T) {
	cmd := New()
	for _, name := range []string{"bind", "unbind"} {
		sub, _, err := cmd.Find([]string{name})
		if err != nil || sub == nil {
			t.Fatalf("subcommand %q not found", name)
		}
		flag := sub.Flags().Lookup("auto-approve")
		assert.NotNil(t, flag, "%s missing --auto-approve flag", name)
	}
}
