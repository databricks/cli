package logdiag_test

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
)

func TestIsolatedContext(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	isolated := logdiag.IsolatedContext(ctx)
	logdiag.SetCollect(isolated, true)
	logdiag.LogError(isolated, errors.New("inner failure"))

	assert.True(t, logdiag.HasError(isolated))
	assert.Len(t, logdiag.FlushCollected(isolated), 1)

	assert.False(t, logdiag.HasError(ctx))
	assert.Empty(t, logdiag.FlushCollected(ctx))
}

func TestLocationsFilledFromPaths(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	var calls []string
	logdiag.SetLocationsOf(ctx, func(p *structpath.PathNode) []diag.Location {
		calls = append(calls, p.String())
		return []diag.Location{{File: p.String(), Line: 1}}
	})

	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "paths only",
		Paths:    structpath.MustParsePaths("a.b", "c"),
	})
	explicit := []diag.Location{{File: "explicit", Line: 7}}
	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   "explicit",
		Paths:     structpath.MustParsePaths("a.b"),
		Locations: explicit,
	})

	got := logdiag.FlushCollected(ctx)
	assert.Len(t, got, 2)
	assert.Equal(t, []string{"a.b", "c"}, calls)
	assert.Equal(t, []diag.Location{{File: "a.b", Line: 1}, {File: "c", Line: 1}}, got[0].Locations)
	assert.Equal(t, explicit, got[1].Locations)
}

func TestLocationsNilResolver(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	logdiag.LogDiag(ctx, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "no resolver",
		Paths:    structpath.MustParsePaths("a.b"),
	})

	got := logdiag.FlushCollected(ctx)
	assert.Len(t, got, 1)
	assert.Empty(t, got[0].Locations)
}
