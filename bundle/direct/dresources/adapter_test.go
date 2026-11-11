package dresources_test

import (
	"context"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourcesWithoutCreate(t *testing.T) {
	for resourceType, resource := range dresources.SupportedResources {
		adapter, err := dresources.NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)
		noCreate := strings.HasSuffix(resourceType, ".grants") || strings.HasSuffix(resourceType, ".permissions")
		assert.Equal(t, !noCreate, adapter.HasDoCreate(), resourceType)
		if noCreate {
			assert.True(t, adapter.HasIDFromState(), resourceType)
			assert.True(t, adapter.HasDoUpdate(), resourceType)
		}
	}
}

func TestResourcesWithoutCreateRequireIdentityAndUpdate(t *testing.T) {
	for _, tc := range []struct {
		missing  string
		resource any
	}{
		{"IDFromState", (*resourceWithoutIdentity)(nil)},
		{"DoUpdate", (*resourceWithoutUpdate)(nil)},
	} {
		t.Run(tc.missing, func(t *testing.T) {
			_, err := dresources.NewAdapter(tc.resource, "test", nil)
			assert.EqualError(t, err, "resources without DoCreate require IDFromState and DoUpdate")
		})
	}
}

type resourceWithoutCreate struct{}

func (*resourceWithoutCreate) PrepareState(state *dresources.GrantsState) *dresources.GrantsState {
	return state
}

func (*resourceWithoutCreate) DoRead(context.Context, string) (*dresources.GrantsState, error) {
	return nil, nil
}

type resourceWithoutIdentity struct{ resourceWithoutCreate }

func (*resourceWithoutIdentity) New(*databricks.WorkspaceClient) *resourceWithoutIdentity {
	return &resourceWithoutIdentity{}
}

func (*resourceWithoutIdentity) DoUpdate(context.Context, string, *dresources.GrantsState, *dresources.PlanEntry) (*dresources.GrantsState, error) {
	return nil, nil
}

type resourceWithoutUpdate struct{ resourceWithoutCreate }

func (*resourceWithoutUpdate) New(*databricks.WorkspaceClient) *resourceWithoutUpdate {
	return &resourceWithoutUpdate{}
}

func (*resourceWithoutUpdate) IDFromState(state *dresources.GrantsState) string {
	return state.FullName
}
