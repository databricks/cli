package dms

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/structs/structdiff"
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

// Deployment renders the metadata the deployment owns.
func (m Metadata) Deployment() bundledeployments.Deployment {
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

	want := m.Deployment()
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
	if !structdiff.IsEqual(want.WorkspaceInfo, current.WorkspaceInfo) {
		stale = append(stale, "workspace_info")
	}
	return strings.Join(stale, ",")
}

// NextVersion is the version number this run will create, given the deployment's most recent
// one. lastVersionID is empty before the deployment has any version.
func NextVersion(lastVersionID string) (int, error) {
	if lastVersionID == "" {
		return 1, nil
	}
	last, err := strconv.Atoi(lastVersionID)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last_version_id %q: %w", lastVersionID, err)
	}
	return last + 1, nil
}

// DeploymentNodeName is the workspace node DMS creates per deployment. Must
// match DeploymentWhsClient.DEPLOYMENT_NODE_NAME on the service side.
const DeploymentNodeName = "resources.deployment.json"

// DeploymentName and VersionName are the two resource-name formats the service uses, so a
// caller only ever passes ids.
func DeploymentName(deploymentID string) string {
	return "deployments/" + deploymentID
}

func VersionName(deploymentID string, version int) string {
	return fmt.Sprintf("deployments/%s/versions/%d", deploymentID, version)
}

// DeploymentIDFromName extracts the deployment ID from a DMS resource name of
// the form "deployments/{deployment_id}".
func DeploymentIDFromName(name string) (string, error) {
	id, ok := strings.CutPrefix(name, DeploymentName(""))
	if !ok || id == "" {
		return "", fmt.Errorf("unexpected deployment name %q from the deployment history service", name)
	}
	return id, nil
}

// DeploymentUpdate builds the deployment carrying exactly the masked fields, empty ones
// included: the service requires every masked field to be present in the body and reads an empty
// value as a clear (a target that stops setting mode clears deployment_mode). ForceSendFields
// keeps those empty values on the wire, which omitempty would drop.
func DeploymentUpdate(metadata Metadata, mask string) bundledeployments.Deployment {
	full := metadata.Deployment()
	var dep bundledeployments.Deployment
	for path := range strings.SplitSeq(mask, ",") {
		switch path {
		case "display_name":
			dep.DisplayName = full.DisplayName
			dep.ForceSendFields = append(dep.ForceSendFields, "DisplayName")
		case "target_name":
			dep.TargetName = full.TargetName
			dep.ForceSendFields = append(dep.ForceSendFields, "TargetName")
		case "deployment_mode":
			dep.DeploymentMode = full.DeploymentMode
			dep.ForceSendFields = append(dep.ForceSendFields, "DeploymentMode")
		case "workspace_info":
			dep.WorkspaceInfo = full.WorkspaceInfo
			dep.ForceSendFields = append(dep.ForceSendFields, "WorkspaceInfo")
		}
	}
	return dep
}
