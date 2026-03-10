//go:build windows

package project

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAtLeastOnePythonInstalled(t *testing.T) {
	ctx := t.Context()
	all, err := DetectInterpreters(ctx)
	assert.NoError(t, err)
	a := all.Latest()
	t.Logf("latest is: %s", a)
	assert.True(t, len(all) > 0)
}

func TestNoInterpretersFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	ctx := t.Context()
	_, err := DetectInterpreters(ctx)
	assert.ErrorIs(t, err, ErrNoPythonInterpreters)
	assert.ErrorContains(t, err, "python.org/downloads")
}
