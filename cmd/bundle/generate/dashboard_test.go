package generate

import (
	"io"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboard_ErrorOnLegacyDashboard(t *testing.T) {
	// Response to a GetStatus request on a path pointing to a legacy dashboard.
	//
	// < HTTP/2.0 400 Bad Request
	// < {
	// <   "error_code": "BAD_REQUEST",
	// <   "message": "dbsqlDashboard is not user-facing."
	// < }

	d := dashboard{
		existingPath: "/path/to/legacy dashboard",
	}

	cfg, _ := httpfixtures.Fixtures{{
		Method:   "GET",
		Resource: "/api/2.0/workspace/get-status?path=/path/to/legacy%20dashboard",
		Status:   400,
		Response: `{"error_code":"BAD_REQUEST","message":"dbsqlDashboard is not user-facing."}`,
	}}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)

	ctx := t.Context()
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)
	b := &bundle.Bundle{}
	b.SetWorkpaceClient(w)

	id := d.resolveID(ctx, b)
	assert.Empty(t, id)

	diags := logdiag.FlushCollected(ctx)
	require.Len(t, diags, 1)
	assert.Equal(t, "dashboard \"legacy dashboard\" is a legacy dashboard", diags[0].Summary)
}

func TestDashboard_LookupFlagsAreMutuallyExclusive(t *testing.T) {
	tests := [][]string{
		{"--existing-path", "/path/to/dashboard", "--existing-id", "f00dcafe"},
		{"--existing-path", "/path/to/dashboard", "--resource", "test_dashboard"},
		{"--existing-id", "f00dcafe", "--resource", "test_dashboard"},
	}

	for _, args := range tests {
		cmd := NewGenerateDashboardCommand()
		cmd.SetArgs(args)
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		err := cmd.Execute()
		assert.ErrorContains(t, err, "none of the others can be")
	}
}
