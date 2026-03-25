package auth

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePositionalArgMatchesProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
		},
	}

	profileName, host, err := resolvePositionalArg(ctx, "logfood", profiler)
	require.NoError(t, err)
	assert.Equal(t, "logfood", profileName)
	assert.Empty(t, host)
}

func TestResolvePositionalArgFallsBackToHost(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
		},
	}

	profileName, host, err := resolvePositionalArg(ctx, "https://other.cloud.databricks.com", profiler)
	require.NoError(t, err)
	assert.Empty(t, profileName)
	assert.Equal(t, "https://other.cloud.databricks.com", host)
}

func TestResolvePositionalArgFallsBackToHostWithDot(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{},
	}

	profileName, host, err := resolvePositionalArg(ctx, "my-workspace.cloud.databricks.com", profiler)
	require.NoError(t, err)
	assert.Empty(t, profileName)
	assert.Equal(t, "my-workspace.cloud.databricks.com", host)
}

func TestResolvePositionalArgErrorsForNonHostNonProfile(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "logfood", Host: "https://logfood.cloud.databricks.com"},
		},
	}

	_, _, err := resolvePositionalArg(ctx, "e2-logfood", profiler)
	assert.ErrorContains(t, err, `no profile named "e2-logfood" found`)
}

func TestResolvePositionalArgHttpPrefix(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{},
	}

	profileName, host, err := resolvePositionalArg(ctx, "http://localhost:8080", profiler)
	require.NoError(t, err)
	assert.Empty(t, profileName)
	assert.Equal(t, "http://localhost:8080", host)
}

func TestResolvePositionalArgEmptyProfiles(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{}

	_, _, err := resolvePositionalArg(ctx, "myprofile", profiler)
	assert.ErrorContains(t, err, `no profile named "myprofile" found`)
}

func TestResolvePositionalArgProfileWithDotInName(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	profiler := profile.InMemoryProfiler{
		Profiles: profile.Profiles{
			{Name: "default.dev", Host: "https://dev.cloud.databricks.com"},
		},
	}

	profileName, host, err := resolvePositionalArg(ctx, "default.dev", profiler)
	require.NoError(t, err)
	assert.Equal(t, "default.dev", profileName)
	assert.Empty(t, host)
}
