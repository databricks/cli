package tfdyn

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/common/types/duration"
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPostgresBranch(t *testing.T) {
	src := resources.PostgresBranch{
		PostgresBranchConfig: resources.PostgresBranchConfig{
			BranchId: "my-branch",
			Parent:   "projects/my-project",
			BranchSpec: postgres.BranchSpec{
				IsProtected: true,
				NoExpiry:    true,
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresBranchConverter{}.Convert(ctx, "my_postgres_branch", vin, out)
	require.NoError(t, err)

	postgresBranch := out.PostgresBranch["my_postgres_branch"]
	assert.Equal(t, map[string]any{
		"branch_id": "my-branch",
		"parent":    "projects/my-project",
		"spec": map[string]any{
			"is_protected": true,
			"no_expiry":    true,
		},
	}, postgresBranch)
}

func TestConvertPostgresBranchWithSourceBranch(t *testing.T) {
	src := resources.PostgresBranch{
		PostgresBranchConfig: resources.PostgresBranchConfig{
			BranchId: "feature-branch",
			Parent:   "projects/my-project",
			BranchSpec: postgres.BranchSpec{
				SourceBranch:    "projects/my-project/branches/main",
				SourceBranchLsn: "0/1234ABCD",
				Ttl:             duration.New(86400 * time.Second),
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresBranchConverter{}.Convert(ctx, "feature_postgres_branch", vin, out)
	require.NoError(t, err)

	postgresBranch := out.PostgresBranch["feature_postgres_branch"]
	assert.Equal(t, map[string]any{
		"branch_id": "feature-branch",
		"parent":    "projects/my-project",
		"spec": map[string]any{
			"source_branch":     "projects/my-project/branches/main",
			"source_branch_lsn": "0/1234ABCD",
			"ttl":               "86400s",
		},
	}, postgresBranch)
}

func TestConvertPostgresBranchMinimal(t *testing.T) {
	src := resources.PostgresBranch{
		PostgresBranchConfig: resources.PostgresBranchConfig{
			BranchId: "minimal-branch",
			Parent:   "projects/minimal-project",
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = postgresBranchConverter{}.Convert(ctx, "minimal_postgres_branch", vin, out)
	require.NoError(t, err)

	postgresBranch := out.PostgresBranch["minimal_postgres_branch"]
	assert.Equal(t, map[string]any{
		"branch_id": "minimal-branch",
		"parent":    "projects/minimal-project",
	}, postgresBranch)
}
