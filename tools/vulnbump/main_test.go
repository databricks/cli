package main

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunEndToEnd drives the whole program over the sample govulncheck output:
// it must issue one `go get` per module (highest fixed version), a single
// `go mod tidy`, and print the linked summary. The toolchain is stubbed so the
// test stays hermetic; the workflow runs the same flow against the real `go`.
func TestRunEndToEnd(t *testing.T) {
	var calls [][]string
	runCmd := func(dir, name string, args ...string) error {
		calls = append(calls, append([]string{dir, name}, args...))
		return nil
	}

	var out strings.Builder
	require.NoError(t, run(".", strings.NewReader(sampleOutput(t)), &out, runCmd))

	assert.Equal(t, [][]string{
		{".", "go", "get", "golang.org/x/crypto@v0.52.0"},
		{".", "go", "get", "golang.org/x/net@v0.55.0"},
		{".", "go", "mod", "tidy"},
	}, calls)

	assert.Contains(t, out.String(), "golang.org/x/crypto → v0.52.0")
	assert.Contains(t, out.String(), "[CVE-2026-39827]")
	assert.Contains(t, out.String(), "[CVE-2026-39827]: https://pkg.go.dev/vuln/GO-2026-5016")
}

func TestRunNoFindings(t *testing.T) {
	calls := 0
	runCmd := func(dir, name string, args ...string) error {
		calls++
		return nil
	}

	var out strings.Builder
	require.NoError(t, run(".", strings.NewReader(""), &out, runCmd))

	// No vulnerabilities means nothing to upgrade and an empty summary, so the
	// workflow's `git diff` check leaves it without opening a PR.
	assert.Zero(t, calls)
	assert.Empty(t, out.String())
}

func TestRunStopsOnError(t *testing.T) {
	runCmd := func(dir, name string, args ...string) error {
		return errors.New("network down")
	}

	err := run(".", strings.NewReader(sampleOutput(t)), io.Discard, runCmd)
	require.Error(t, err)
}
