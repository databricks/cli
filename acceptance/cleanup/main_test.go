package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePrefix(t *testing.T) {
	t.Run("CI default", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_ID", "123")
		prefix, err := resolvePrefix(t.Context(), "")
		require.NoError(t, err)
		assert.Equal(t, "ci123x", prefix)
	})

	t.Run("local requires explicit prefix", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_ID", "")
		_, err := resolvePrefix(t.Context(), "")
		assert.EqualError(t, err, "-prefix is required outside CI; use the exact local prefix printed by the acceptance run")
	})

	t.Run("rejects generic prefix", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_ID", "")
		_, err := resolvePrefix(t.Context(), "local")
		assert.ErrorContains(t, err, "missing its base36 timestamp or random suffix")
	})

	t.Run("rejects prefix from another CI run", func(t *testing.T) {
		t.Setenv("GITHUB_RUN_ID", "123")
		_, err := resolvePrefix(t.Context(), "ci456x")
		assert.ErrorContains(t, err, "does not match CI run prefix")
	})
}
