package acceptance_test

import (
	"testing"

	"github.com/databricks/cli/acceptance/internal"
	"github.com/stretchr/testify/assert"
)

func TestDbrSkipReason(t *testing.T) {
	prev := Dbr
	Dbr = true
	t.Cleanup(func() { Dbr = prev })

	unset := &internal.TestConfig{Cloud: new(true)}
	optIn := &internal.TestConfig{Cloud: new(true), RunsOnDbr: new(true)}
	optOut := &internal.TestConfig{Cloud: new(true), RunsOnDbr: new(false)}

	t.Run("local runs are opt-out", func(t *testing.T) {
		t.Setenv("CLOUD_ENV", "")
		assert.Empty(t, getSkipReason(unset, "test.toml", false))
		assert.Empty(t, getSkipReason(optIn, "test.toml", false))
		assert.Contains(t, getSkipReason(optOut, "test.toml", false), "RunsOnDbr")
	})

	t.Run("cloud runs are opt-in", func(t *testing.T) {
		t.Setenv("CLOUD_ENV", "aws")
		assert.Contains(t, getSkipReason(unset, "test.toml", false), "RunsOnDbr")
		assert.Empty(t, getSkipReason(optIn, "test.toml", false))
		assert.Contains(t, getSkipReason(optOut, "test.toml", false), "RunsOnDbr")
	})
}
