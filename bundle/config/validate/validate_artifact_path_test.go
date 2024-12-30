package validate_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestValidateArtifactPathWithVolumeInBundle(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/catalogN/schemaN/volumeN/abc",
			},
			Resources: config.Resources{
				Volumes: map[string]*resources.Volume{
					"foo": {
						CreateVolumeRequestContent: &catalog.CreateVolumeRequestContent{
							CatalogName: "catalogN",
							Name:        "volumeN",
							VolumeType:  "MANAGED",
							SchemaName:  "schemaN",
						},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	bundletest.SetLocation(b, "resources.volumes.foo", []dyn.Location{{File: "file", Line: 2, Column: 2}})

	ctx := context.Background()
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockGrantsAPI()
	api.EXPECT().GetEffectiveBySecurableTypeAndFullName(mock.Anything, catalog.SecurableTypeVolume, "catalogN.schemaN.volumeN").Return(nil, &apierr.APIError{
		StatusCode: 404,
	})
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), validate.ValidateArtifactPath())
	assert.Equal(t, diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  "volume catalogN.schemaN.volumeN does not exist",
		Locations: []dyn.Location{
			{File: "file", Line: 1, Column: 1},
			{File: "file", Line: 2, Column: 2},
		},
		Paths: []dyn.Path{
			dyn.MustPathFromString("workspace.artifact_path"),
			dyn.MustPathFromString("resources.volumes.foo"),
		},
		Detail: `You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.`,
	}}, diags)
}

func TestValidateArtifactPath(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				ArtifactPath: "/Volumes/catalogN/schemaN/volumeN/abc",
			},
		},
	}

	bundletest.SetLocation(b, "workspace.artifact_path", []dyn.Location{{File: "file", Line: 1, Column: 1}})
	assertDiags := func(t *testing.T, diags diag.Diagnostics, expected string) {
		assert.Len(t, diags, 1)
		assert.Equal(t, diag.Diagnostics{{
			Severity:  diag.Error,
			Summary:   expected,
			Locations: []dyn.Location{{File: "file", Line: 1, Column: 1}},
			Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
		}}, diags)
	}

	wrapPrivileges := func(privileges ...catalog.Privilege) *catalog.EffectivePermissionsList {
		perms := &catalog.EffectivePermissionsList{}
		for _, p := range privileges {
			perms.PrivilegeAssignments = append(perms.PrivilegeAssignments, catalog.EffectivePrivilegeAssignment{
				Privileges: []catalog.EffectivePrivilege{{Privilege: p}},
			})
		}
		return perms
	}

	rb := bundle.ReadOnly(b)
	ctx := context.Background()

	tcases := []struct {
		err             error
		permissions     *catalog.EffectivePermissionsList
		expectedSummary string
	}{
		{
			err: &apierr.APIError{
				StatusCode: 403,
				Message:    "User does not have USE SCHEMA on Schema 'catalogN.schemaN'",
			},
			expectedSummary: "cannot access volume catalogN.schemaN.volumeN: User does not have USE SCHEMA on Schema 'catalogN.schemaN'",
		},
		{
			err: &apierr.APIError{
				StatusCode: 404,
			},
			expectedSummary: "volume catalogN.schemaN.volumeN does not exist",
		},
		{
			err: &apierr.APIError{
				StatusCode: 500,
				Message:    "Internal Server Error",
			},
			expectedSummary: "could not fetch grants for volume catalogN.schemaN.volumeN: Internal Server Error",
		},
		{
			permissions: wrapPrivileges(catalog.PrivilegeAllPrivileges),
		},
		{
			permissions:     wrapPrivileges(catalog.PrivilegeApplyTag, catalog.PrivilegeManage),
			expectedSummary: "user does not have WRITE_VOLUME grant on volume catalogN.schemaN.volumeN",
		},
		{
			permissions:     wrapPrivileges(catalog.PrivilegeWriteVolume),
			expectedSummary: "user does not have READ_VOLUME grant on volume catalogN.schemaN.volumeN",
		},
		{
			permissions: wrapPrivileges(catalog.PrivilegeWriteVolume, catalog.PrivilegeReadVolume),
		},
	}

	for _, tc := range tcases {
		m := mocks.NewMockWorkspaceClient(t)
		api := m.GetMockGrantsAPI()
		api.EXPECT().GetEffectiveBySecurableTypeAndFullName(mock.Anything, catalog.SecurableTypeVolume, "catalogN.schemaN.volumeN").Return(tc.permissions, tc.err)
		b.SetWorkpaceClient(m.WorkspaceClient)

		diags := bundle.ApplyReadOnly(ctx, rb, validate.ValidateArtifactPath())
		if tc.expectedSummary != "" {
			assertDiags(t, diags, tc.expectedSummary)
		} else {
			assert.Len(t, diags, 0)
		}
	}
}
