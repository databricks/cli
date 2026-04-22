package phases_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/direct"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// driftFakeClient extends fakeDirectClient with GetCatalog so the drift
// comparator has something to read. Other Get* fall through to the base
// fake's nil-returning behavior, which the comparator treats as "no live
// resource exists" — harmless for fixtures that only record a catalog.
type driftFakeClient struct {
	fakeDirectClient
	catalogs    map[string]*catalog.CatalogInfo
	catalogErrs map[string]error
}

func (c *driftFakeClient) GetCatalog(_ context.Context, name string) (*catalog.CatalogInfo, error) {
	if err := c.catalogErrs[name]; err != nil {
		return nil, err
	}
	return c.catalogs[name], nil
}

func TestDriftReturnsEmptyReportOnMissingState(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	factory := func(_ context.Context, _ *ucm.Ucm) (direct.Client, error) {
		return &driftFakeClient{}, nil
	}
	report := phases.Drift(ctx, f.u, phases.Options{DirectClientFactory: factory})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	require.NotNil(t, report)
	assert.False(t, report.HasDrift())
}

func TestDriftReportsFieldMismatch(t *testing.T) {
	f := newFixture(t)
	seedDirectState(t, f.u, &direct.State{
		Version:  direct.StateVersion,
		Catalogs: map[string]*direct.CatalogState{"sales": {Name: "sales", Comment: "sales data"}},
	})

	client := &driftFakeClient{
		catalogs: map[string]*catalog.CatalogInfo{
			"sales": {Name: "sales", Comment: "drifted"},
		},
	}
	factory := func(_ context.Context, _ *ucm.Ucm) (direct.Client, error) { return client, nil }

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	report := phases.Drift(ctx, f.u, phases.Options{DirectClientFactory: factory})

	require.False(t, logdiag.HasError(ctx), "unexpected errors: %v", logdiag.FlushCollected(ctx))
	require.NotNil(t, report)
	require.True(t, report.HasDrift())
	require.Len(t, report.Drift, 1)
	assert.Equal(t, "resources.catalogs.sales", report.Drift[0].Key)
}

func TestDriftBailsOnClientFactoryError(t *testing.T) {
	f := newFixture(t)
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	factory := func(_ context.Context, _ *ucm.Ucm) (direct.Client, error) {
		return nil, errors.New("boom")
	}

	report := phases.Drift(ctx, f.u, phases.Options{DirectClientFactory: factory})

	assert.Nil(t, report)
	require.True(t, logdiag.HasError(ctx))
}

func TestDriftPropagatesComparatorError(t *testing.T) {
	f := newFixture(t)
	seedDirectState(t, f.u, &direct.State{
		Version:  direct.StateVersion,
		Catalogs: map[string]*direct.CatalogState{"sales": {Name: "sales"}},
	})

	client := &driftFakeClient{catalogErrs: map[string]error{"sales": errors.New("500 internal")}}
	factory := func(_ context.Context, _ *ucm.Ucm) (direct.Client, error) { return client, nil }

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	report := phases.Drift(ctx, f.u, phases.Options{DirectClientFactory: factory})

	assert.Nil(t, report)
	require.True(t, logdiag.HasError(ctx))
}

// seedDirectState writes a direct.State file at the canonical location
// phases.Drift reads from so tests see a non-empty state.
func seedDirectState(t *testing.T, u *ucm.Ucm, state *direct.State) {
	t.Helper()
	require.NoError(t, direct.SaveState(direct.StatePath(u), state))
}
