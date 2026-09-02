package dms

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

// VersionType identifies the kind of deployment a version records.
type VersionType = bundledeployments.VersionType

const (
	VersionTypeDeploy  VersionType = bundledeployments.VersionTypeVersionTypeDeploy
	VersionTypeDestroy VersionType = bundledeployments.VersionTypeVersionTypeDestroy
)

// Metadata is what a version records about the bundle, its source and where it
// landed. The service copies these onto the deployment, so they describe it as of
// its most recent version.
type Metadata struct {
	// DisplayName is the bundle's name, which the deployment is listed under.
	DisplayName string
	// TargetName is the bundle target that was deployed.
	TargetName string
	// Mode is the bundle target's mode, empty when the target sets none.
	Mode      bundledeployments.DeploymentMode
	Workspace *bundledeployments.WorkspaceInfo
}

// deploymentFields are the deployment's own metadata, in the order a mask lists them. Git is not
// among them: the service derives the deployment's from the version that carried it.
var deploymentFields = []string{"display_name", "target_name", "deployment_mode", "workspace_info"}

// sameWorkspaceInfo compares the paths alone. The SDK records which fields a response carried in
// ForceSendFields, so a record read back never deep-equals one built here, and comparing the
// structs whole would report every run as a change.
func sameWorkspaceInfo(want, current *bundledeployments.WorkspaceInfo) bool {
	if want == nil || current == nil {
		return want == nil && current == nil
	}

	a, b := *want, *current
	a.ForceSendFields, b.ForceSendFields = nil, nil
	return reflect.DeepEqual(a, b)
}

// deployment renders the metadata the deployment owns.
func (m Metadata) deployment() bundledeployments.Deployment {
	return bundledeployments.Deployment{
		DisplayName:    m.DisplayName,
		TargetName:     m.TargetName,
		DeploymentMode: m.Mode,
		WorkspaceInfo:  m.Workspace,
	}
}

// StaleFields returns the mask that brings current up to m, empty when the deployment already
// says what this run would say. current is nil before the first recorded deploy.
func (m Metadata) StaleFields(current *bundledeployments.Deployment) string {
	if current == nil {
		return strings.Join(deploymentFields, ",")
	}

	want := m.deployment()
	var stale []string
	if want.DisplayName != current.DisplayName {
		stale = append(stale, "display_name")
	}
	if want.TargetName != current.TargetName {
		stale = append(stale, "target_name")
	}
	if want.DeploymentMode != current.DeploymentMode {
		stale = append(stale, "deployment_mode")
	}
	if !sameWorkspaceInfo(want.WorkspaceInfo, current.WorkspaceInfo) {
		stale = append(stale, "workspace_info")
	}
	return strings.Join(stale, ",")
}

// NextVersion is the version number this run will create, given the deployment's most recent
// one. lastVersionID is empty before the deployment has any version.
func NextVersion(lastVersionID string) (int64, error) {
	if lastVersionID == "" {
		return 1, nil
	}
	last, err := strconv.ParseInt(lastVersionID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last_version_id %q: %w", lastVersionID, err)
	}
	return last + 1, nil
}
