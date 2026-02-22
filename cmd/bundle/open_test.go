package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveOpenArgument_NoArgs_NonInteractive(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	b := &bundle.Bundle{}

	_, err := resolveOpenArgument(ctx, b, nil)
	require.Error(t, err)
	assert.ErrorContains(t, err, "expected a KEY of the resource to open")
}

func TestResolveOpenArgument_ExactMatch(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job":   {JobSettings: jobs.JobSettings{Name: "My Job"}},
					"my_job_2": {JobSettings: jobs.JobSettings{Name: "My Job 2"}},
				},
			},
		},
	}

	key, err := resolveOpenArgument(ctx, b, []string{"my_job"})
	require.NoError(t, err)
	assert.Equal(t, "my_job", key)
}

func TestResolveOpenArgument_PrefixSingleMatch(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo_job": {JobSettings: jobs.JobSettings{Name: "Foo Job"}},
					"bar_job": {JobSettings: jobs.JobSettings{Name: "Bar Job"}},
				},
			},
		},
	}

	key, err := resolveOpenArgument(ctx, b, []string{"foo"})
	require.NoError(t, err)
	assert.Equal(t, "foo_job", key)
}

func TestResolveOpenArgument_PrefixMultipleMatches_NonInteractive(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job_1": {JobSettings: jobs.JobSettings{Name: "My Job 1"}},
					"my_job_2": {JobSettings: jobs.JobSettings{Name: "My Job 2"}},
					"other":    {JobSettings: jobs.JobSettings{Name: "Other"}},
				},
			},
		},
	}

	_, err := resolveOpenArgument(ctx, b, []string{"my_"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "multiple resources match prefix")
	assert.ErrorContains(t, err, "my_job_1")
	assert.ErrorContains(t, err, "my_job_2")
}

func TestResolveOpenArgument_PrefixNoMatch(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {JobSettings: jobs.JobSettings{Name: "Foo"}},
				},
			},
		},
	}

	// No prefix match; returns the arg as-is.
	key, err := resolveOpenArgument(ctx, b, []string{"zzz"})
	require.NoError(t, err)
	assert.Equal(t, "zzz", key)
}
