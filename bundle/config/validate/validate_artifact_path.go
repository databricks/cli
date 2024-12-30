package validate

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

type validateArtifactPath struct{}

func ValidateArtifactPath() bundle.ReadOnlyMutator {
	return &validateArtifactPath{}
}

func (v *validateArtifactPath) Name() string {
	return "validate:artifact_paths"
}

func (v *validateArtifactPath) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	// We only validate UC Volumes paths right now.
	if !libraries.IsVolumesPath(rb.Config().Workspace.ArtifactPath) {
		return nil
	}

	catalogName, schemaName, volumeName, err := libraries.ExtractVolumeFromPath(rb.Config().Workspace.ArtifactPath)
	if err != nil {
		return diag.FromErr(err)
	}
	volumeFullName := fmt.Sprintf("%s.%s.%s", catalogName, schemaName, volumeName)
	w := rb.WorkspaceClient()
	p, err := w.Grants.GetEffectiveBySecurableTypeAndFullName(ctx, catalog.SecurableTypeVolume, volumeFullName)

	wrapErrorMsg := func(s string) diag.Diagnostics {
		return diag.Diagnostics{
			{
				Summary:   s,
				Severity:  diag.Error,
				Locations: rb.Config().GetLocations("workspace.artifact_path"),
				Paths:     []dyn.Path{dyn.MustPathFromString("workspace.artifact_path")},
			},
		}
	}
	if errors.Is(err, apierr.ErrPermissionDenied) {
		return wrapErrorMsg(fmt.Sprintf("cannot access volume %s: %s", volumeFullName, err))
	}
	if errors.Is(err, apierr.ErrNotFound) {
		path, locations, ok := libraries.FindVolumeInBundle(rb.Config(), catalogName, schemaName, volumeName)
		if !ok {
			return wrapErrorMsg(fmt.Sprintf("volume %s does not exist", volumeFullName))
		}

		// If the volume is defined in the bundle, provide a more helpful error diagnostic,
		// with more details and location information.
		return diag.Diagnostics{{
			Summary:  fmt.Sprintf("volume %s does not exist", volumeFullName),
			Severity: diag.Error,
			Detail: `You are using a volume in your artifact_path that is managed by
this bundle but which has not been deployed yet. Please first deploy
the volume using 'bundle deploy' and then switch over to using it in
the artifact_path.`,
			Locations: slices.Concat(rb.Config().GetLocations("workspace.artifact_path"), locations),
			Paths:     append([]dyn.Path{dyn.MustPathFromString("workspace.artifact_path")}, path),
		}}

	}
	if err != nil {
		return wrapErrorMsg(fmt.Sprintf("could not fetch grants for volume %s: %s", volumeFullName, err))
	}

	allPrivileges := []catalog.Privilege{}
	for _, assignments := range p.PrivilegeAssignments {
		for _, privilege := range assignments.Privileges {
			allPrivileges = append(allPrivileges, privilege.Privilege)
		}
	}

	// UC Volumes have the following privileges: [READ_VOLUME, WRITE_VOLUME, MANAGE, ALL_PRIVILEGES, APPLY TAG]
	// The user needs to have either WRITE_VOLUME or ALL_PRIVILEGES to write to the volume.
	canWrite := slices.Contains(allPrivileges, catalog.PrivilegeWriteVolume) || slices.Contains(allPrivileges, catalog.PrivilegeAllPrivileges)
	if !canWrite {
		return wrapErrorMsg(fmt.Sprintf("user does not have WRITE_VOLUME grant on volume %s", volumeFullName))
	}

	// READ_VOLUME is implied since the user was able to fetch the associated grants with the volume.
	// We still add this explicit check out of caution incase the API behavior changes in the future.
	canRead := slices.Contains(allPrivileges, catalog.PrivilegeReadVolume) || slices.Contains(allPrivileges, catalog.PrivilegeAllPrivileges)
	if !canRead {
		return wrapErrorMsg(fmt.Sprintf("user does not have READ_VOLUME grant on volume %s", volumeFullName))
	}

	return nil
}
