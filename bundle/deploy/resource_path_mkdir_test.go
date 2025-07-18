package deploy

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResourcePathMkdir_NoDashboards(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			// No dashboards configured
			Resources: config.Resources{},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	// No API calls should be made when there are no dashboards
	diags := bundle.Apply(context.Background(), b, ResourcePathMkdir())
	require.Empty(t, diags)
}

func TestResourcePathMkdir_PathExists(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"test_dashboard": {},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()

	// Mock that the path already exists
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com/resources").Return(&workspace.ObjectInfo{
		ObjectId: 1234,
	}, nil)

	b.SetWorkpaceClient(m.WorkspaceClient)

	// Should not try to create directory since it already exists
	diags := bundle.Apply(context.Background(), b, ResourcePathMkdir())
	require.Empty(t, diags)
}

func TestResourcePathMkdir_PathDoesNotExist_CreatesDirectory(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"test_dashboard": {},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()

	// Mock that the path doesn't exist (404 error)
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com/resources").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})

	// Mock successful directory creation
	api.EXPECT().MkdirsByPath(mock.Anything, "/Workspace/Users/foo@bar.com/resources").Return(nil)

	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(context.Background(), b, ResourcePathMkdir())
	require.Empty(t, diags)
}

func TestResourcePathMkdir_GetStatusError_NonNotFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"test_dashboard": {},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()

	// Mock that GetStatusByPath returns a non-404 error
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com/resources").Return(nil, &apierr.APIError{
		StatusCode: 403,
		ErrorCode:  "PERMISSION_DENIED",
		Message:    "Could not get status",
	})

	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(context.Background(), b, ResourcePathMkdir())
	require.Len(t, diags, 1)
	require.Equal(t, diags[0].Summary, "Could not get status")
}

func TestResourcePathMkdir_MkdirsFails(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"test_dashboard": {},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()

	// Mock that the path doesn't exist (404 error)
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com/resources").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
		Message:    "Could not get status",
	})

	// Mock that directory creation fails
	api.EXPECT().MkdirsByPath(mock.Anything, "/Workspace/Users/foo@bar.com/resources").Return(&apierr.APIError{
		StatusCode: 403,
		ErrorCode:  "PERMISSION_DENIED",
		Message:    "Could not create directory",
	})

	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(context.Background(), b, ResourcePathMkdir())
	require.Len(t, diags, 1)
	require.Equal(t, diags[0].Summary, "Could not create directory")
}
