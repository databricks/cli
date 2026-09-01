package dresources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

// librariesWaitTimeout bounds how long we poll for libraries to finish installing.
const librariesWaitTimeout = 15 * time.Minute

// clusterWaitTimeout bounds how long we poll for a cluster to reach its target
// state (RUNNING/TERMINATED) after create, edit, or start/stop. Provisioning can
// legitimately take longer than 15 minutes on capacity-constrained workspaces
// (a cluster stays PENDING with "Finding instances for new nodes"), so we allow
// 30 minutes before giving up. Terminal states still halt immediately.
const clusterWaitTimeout = 30 * time.Minute

// ClusterState is the state type for Cluster resources. It extends compute.ClusterSpec with
// lifecycle settings and the cluster ID.
// ClusterId is written to state by DoCreate/DoUpdate for informational purposes; it is not
// used in diff computation (neither PrepareState nor RemapState set it).
type ClusterState struct {
	compute.ClusterSpec

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`

	// Libraries are installed via the Libraries API, not the cluster spec, and managed as
	// part of the cluster (see reconcileLibraries and the install in WaitAfterCreate).
	Libraries []compute.Library `json:"libraries,omitempty"`
}

// Custom marshalers needed because embedded compute.ClusterSpec has its own MarshalJSON
// which would otherwise take over and ignore the additional fields.
func (s *ClusterState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s ClusterState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// ClusterRemote extends compute.ClusterDetails with a synthetic Lifecycle field so that
// RemoteType satisfies TestRemoteSuperset (every field in ClusterState exists in ClusterRemote).
// Lifecycle.Started is populated by DoRead from the cluster's running state.
//
// ApplyPolicyDefaultValues is promoted to the top level because the cluster GET API returns it
// only under .spec (a snapshot of the create/edit settings), never at the top level. DoRead
// copies it up so RemapState stays a dumb copy and the field participates in normal drift
// detection instead of being suppressed as missing_in_remote.
type ClusterRemote struct {
	compute.ClusterDetails
	ApplyPolicyDefaultValues bool            `json:"apply_policy_default_values,omitempty"`
	Lifecycle                *StateLifecycle `json:"lifecycle,omitempty"`

	// Libraries is populated by DoRead from the Libraries cluster-status API (the cluster
	// GET does not return installed libraries), so it participates in drift detection.
	Libraries []compute.Library `json:"libraries,omitempty"`
}

func (r *ClusterRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, r)
}

func (r ClusterRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(r)
}

type ResourceCluster struct {
	client *databricks.WorkspaceClient
}

func (r *ResourceCluster) New(client *databricks.WorkspaceClient) any {
	return &ResourceCluster{
		client: client,
	}
}

func (r *ResourceCluster) PrepareState(input *resources.Cluster) *ClusterState {
	s := &ClusterState{
		ClusterSpec: input.ClusterSpec,
		Lifecycle:   nil,
		Libraries:   input.Libraries,
	}
	if input.Lifecycle != nil && input.Lifecycle.Started != nil {
		s.Lifecycle = &StateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote ClusterRemote to ClusterState for diff comparison.
// Started is derived from cluster state so the planner can detect start/stop changes.
func (r *ResourceCluster) RemapState(input *ClusterRemote) *ClusterState {
	started := input.State == compute.StateRunning
	spec := &ClusterState{
		ClusterSpec: compute.ClusterSpec{
			ApplyPolicyDefaultValues:   input.ApplyPolicyDefaultValues,
			Autoscale:                  input.Autoscale,
			AutoterminationMinutes:     input.AutoterminationMinutes,
			AwsAttributes:              input.AwsAttributes,
			AzureAttributes:            input.AzureAttributes,
			ClusterLogConf:             input.ClusterLogConf,
			ClusterName:                input.ClusterName,
			CustomTags:                 input.CustomTags,
			DataSecurityMode:           input.DataSecurityMode,
			DependencyMode:             input.DependencyMode,
			DockerImage:                input.DockerImage,
			DriverInstancePoolId:       input.DriverInstancePoolId,
			DriverNodeTypeId:           input.DriverNodeTypeId,
			DriverNodeTypeFlexibility:  input.DriverNodeTypeFlexibility,
			EnableElasticDisk:          input.EnableElasticDisk,
			EnableLocalDiskEncryption:  input.EnableLocalDiskEncryption,
			GcpAttributes:              input.GcpAttributes,
			InitScripts:                input.InitScripts,
			InstancePoolId:             input.InstancePoolId,
			IsSingleNode:               input.IsSingleNode,
			Kind:                       input.Kind,
			NodeTypeId:                 input.NodeTypeId,
			NumWorkers:                 input.NumWorkers,
			PolicyId:                   input.PolicyId,
			RemoteDiskThroughput:       input.RemoteDiskThroughput,
			RuntimeEngine:              input.RuntimeEngine,
			SingleUserName:             input.SingleUserName,
			SparkConf:                  input.SparkConf,
			SparkEnvVars:               input.SparkEnvVars,
			SparkVersion:               input.SparkVersion,
			SshPublicKeys:              input.SshPublicKeys,
			TotalInitialRemoteDiskSize: input.TotalInitialRemoteDiskSize,
			UseMlRuntime:               input.UseMlRuntime,
			WorkloadType:               input.WorkloadType,
			WorkerNodeTypeFlexibility:  input.WorkerNodeTypeFlexibility,
			ForceSendFields:            utils.FilterFields[compute.ClusterSpec](input.ForceSendFields),
		},
		Lifecycle: &StateLifecycle{Started: &started},
		Libraries: input.Libraries,
	}
	return spec
}

func (r *ResourceCluster) DoRead(ctx context.Context, id string) (*ClusterRemote, error) {
	details, err := r.client.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		return nil, err
	}
	remote := &ClusterRemote{
		ClusterDetails:           *details,
		ApplyPolicyDefaultValues: false,
		Lifecycle:                nil,
		Libraries:                nil,
	}
	// The GET response carries apply_policy_default_values only under .spec (a snapshot of the
	// create/edit settings), not at the top level. Promote it so RemapState is a dumb copy.
	if details.Spec != nil {
		remote.ApplyPolicyDefaultValues = details.Spec.ApplyPolicyDefaultValues
	}

	switch details.State {
	case compute.StateRunning:
		started := true
		remote.Lifecycle = &StateLifecycle{Started: &started}
	case compute.StateTerminated:
		started := false
		remote.Lifecycle = &StateLifecycle{Started: &started}
	default:
		remote.Lifecycle = nil
	}

	libraries, err := r.readLibraries(ctx, id)
	if err != nil {
		return nil, err
	}
	remote.Libraries = libraries
	return remote, nil
}

// readLibraries returns the bundle-managed libraries installed on the cluster.
// https://docs.databricks.com/api/workspace/libraries/clusterstatus
func (r *ResourceCluster) readLibraries(ctx context.Context, id string) ([]compute.Library, error) {
	statuses, err := r.client.Libraries.ClusterStatusByClusterId(ctx, id)
	if err != nil {
		return nil, err
	}
	var libraries []compute.Library
	for _, s := range statuses.LibraryStatuses {
		// Libraries set for all clusters via the UI are not managed by the bundle; a library
		// pending uninstall on restart is on its way out. Skip both.
		if s.Library == nil || s.IsLibraryForAllClusters || s.Status == compute.LibraryInstallStatusUninstallOnRestart {
			continue
		}
		libraries = append(libraries, *s.Library)
	}
	return libraries, nil
}

func (r *ResourceCluster) DoCreate(ctx context.Context, config *ClusterState) (string, *ClusterRemote, error) {
	wait, err := r.client.Clusters.Create(ctx, makeCreateCluster(&config.ClusterSpec))
	if err != nil {
		return "", nil, err
	}
	return wait.ClusterId, nil, nil
}

// hasClusterSpecChanges reports whether the plan entry changes a Cluster Edit API field —
// anything other than lifecycle (start/stop) and libraries (handled via the Libraries API).
func hasClusterSpecChanges(entry *PlanEntry) bool {
	for field, change := range entry.Changes {
		if change.Action == deployplan.Skip {
			continue
		}
		node, err := structpath.ParsePath(field)
		if err != nil {
			continue
		}
		top, _ := node.Prefix(1).StringKey()
		if top != "lifecycle" && top != "libraries" {
			return true
		}
	}
	return false
}

func (r *ResourceCluster) DoUpdate(ctx context.Context, id string, config *ClusterState, entry *PlanEntry) (*ClusterRemote, error) {
	edited := hasClusterSpecChanges(entry)
	if edited {
		// Same retry as in TF provider logic
		// https://github.com/databricks/terraform-provider-databricks/blob/3eecd0f90cf99d7777e79a3d03c41f9b2aafb004/clusters/resource_cluster.go#L624
		_, err := retries.Poll(ctx, clusterWaitTimeout, func() (*compute.WaitGetClusterRunning[struct{}], *retries.Err) {
			wait, err := r.client.Clusters.Edit(ctx, makeEditCluster(id, &config.ClusterSpec))
			if err == nil {
				return wait, nil
			}

			// Only Running and Terminated clusters can be modified. In particular, autoscaling clusters cannot be modified
			// while the resizing is ongoing. We retry in this case. Scaling can take several minutes.
			if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.ErrorCode == "INVALID_STATE" {
				return nil, retries.Continues(fmt.Sprintf("cluster %s cannot be modified in its current state: %s", id, apiErr.Message))
			}
			return nil, retries.Halt(err)
		})
		if err != nil {
			return nil, err
		}
	}

	// TODO(#1860): a local whl/jar whose workspace path is unchanged but whose contents
	// changed (same name+version, non-dev mode) is not detected here, so no restart fires.
	// Dev mode handles this via patchwheel (a source-derived version bump); a general fix
	// needs a source hash tracked in state. Hashing the built wheel is unsafe — the zip
	// embeds mtimes, so a rebuild would churn a restart every deploy.
	if entry.Changes.HasChange(librariesPath) {
		if err := r.reconcileLibraries(ctx, id, config.Libraries, entry); err != nil {
			return nil, err
		}
		// A cluster edit restarts the cluster on its own, which applies the library change.
		// Without an edit we restart so the change takes effect on a running cluster.
		if !edited {
			if err := r.restartIfRunning(ctx, id); err != nil {
				return nil, err
			}
			if err := r.waitForInstall(ctx, id, config.Libraries); err != nil {
				return nil, err
			}
		}
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	desiredStarted := *config.Lifecycle.Started
	alreadyRunning := remoteClusterIsRunning(entry)
	if desiredStarted && !alreadyRunning {
		// lifecycle.started=true: fire Start; WaitAfterUpdate polls for RUNNING.
		_, err := r.client.Clusters.Start(ctx, compute.StartCluster{ClusterId: id})
		return nil, err
	} else if !desiredStarted && alreadyRunning {
		// lifecycle.started=false: fire Delete; WaitAfterUpdate polls for TERMINATED.
		// Note: Delete terminates the cluster; permanent removal is a separate API (permanent-delete).
		_, err := r.client.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: id})
		return nil, err
	}

	return nil, nil
}

// WaitAfterUpdate waits for the cluster to reach the desired lifecycle state after DoUpdate.
func (r *ResourceCluster) WaitAfterUpdate(ctx context.Context, id string, config *ClusterState) (*ClusterRemote, error) {
	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	if *config.Lifecycle.Started {
		_, err := r.client.Clusters.WaitGetClusterRunning(ctx, id, clusterWaitTimeout, nil)
		return nil, err
	}

	_, err := r.client.Clusters.WaitGetClusterTerminated(ctx, id, clusterWaitTimeout, nil)
	return nil, err
}

// WaitAfterCreate waits for the cluster to reach RUNNING state (clusters always start on creation).
// When lifecycle.started=false, it then terminates the cluster.
func (r *ResourceCluster) WaitAfterCreate(ctx context.Context, id string, config *ClusterState) (*ClusterRemote, error) {
	// Always wait for RUNNING first: clusters start in PENDING state and must be polled.
	_, err := r.client.Clusters.WaitGetClusterRunning(ctx, id, clusterWaitTimeout, nil)
	if err != nil {
		return nil, err
	}

	// Install libraries once the cluster is running. A freshly-created cluster has no
	// attached sessions, so the install applies live without a restart.
	if len(config.Libraries) > 0 {
		err = r.client.Libraries.Install(ctx, compute.InstallLibraries{ClusterId: id, Libraries: config.Libraries})
		if err != nil {
			return nil, err
		}
		if err := r.waitForInstall(ctx, id, config.Libraries); err != nil {
			return nil, err
		}
	}

	if config.Lifecycle != nil && config.Lifecycle.Started != nil && !*config.Lifecycle.Started {
		// started=false: terminate the cluster after it reaches RUNNING.
		// Note: Delete terminates the cluster; permanent removal is a separate API (permanent-delete).
		deleteWaiter, err := r.client.Clusters.Delete(ctx, compute.DeleteCluster{ClusterId: id})
		if err != nil {
			return nil, err
		}
		_, err = deleteWaiter.GetWithTimeout(clusterWaitTimeout)
		return nil, err
	}

	return nil, nil
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *ClusterState, entry *PlanEntry) error {
	_, err := r.client.Clusters.Resize(ctx, compute.ResizeCluster{
		ClusterId:       id,
		NumWorkers:      config.NumWorkers,
		Autoscale:       config.Autoscale,
		ForceSendFields: utils.FilterFields[compute.ResizeCluster](config.ForceSendFields),
	})
	if err == nil {
		return nil
	}

	apiErr, ok := errors.AsType[*apierr.APIError](err)
	if !ok || apiErr.ErrorCode != "INVALID_STATE" {
		return err
	}

	// Cluster is not running; fall back to the full clusters/edit path.
	log.Debugf(ctx, "cluster %s: resize returned INVALID_STATE (%s), falling back to edit", id, err)
	_, err = r.DoUpdate(ctx, id, config, entry)
	return err
}

func (r *ResourceCluster) DoDelete(ctx context.Context, id string, _ *ClusterState) error {
	return r.client.Clusters.PermanentDeleteByClusterId(ctx, id)
}

func (r *ResourceCluster) OverrideChangeDesc(ctx context.Context, p *structpath.PathNode, change *ChangeDesc, remoteState *ClusterRemote) error {
	path := p.Prefix(1).String()

	// Remaining overrides only apply to Update actions.
	if change.Action != deployplan.Update {
		return nil
	}

	switch path {
	case "data_security_mode":
		// We do change skip here in the same way TF provider does suppress diff if the alias is used.
		// https://github.com/databricks/terraform-provider-databricks/blob/main/clusters/resource_cluster.go#L109-L117
		if change.New == compute.DataSecurityModeDataSecurityModeStandard && change.Remote == compute.DataSecurityModeUserIsolation && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeDedicated && change.Remote == compute.DataSecurityModeSingleUser && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == compute.DataSecurityModeDataSecurityModeAuto && (change.Remote == compute.DataSecurityModeSingleUser || change.Remote == compute.DataSecurityModeUserIsolation) && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		}

	case "num_workers", "autoscale":
		if remoteState != nil && remoteState.State == compute.StateRunning {
			change.Action = deployplan.Resize
		}
	}
	return nil
}

// remoteClusterIsRunning reads the cluster running state from the plan entry's remote state.
func remoteClusterIsRunning(entry *PlanEntry) bool {
	if entry.RemoteState == nil {
		return false
	}
	remote, ok := entry.RemoteState.(*ClusterRemote)
	if !ok {
		return false
	}
	return remote.State == compute.StateRunning
}

func makeCreateCluster(config *compute.ClusterSpec) compute.CreateCluster {
	create := compute.CreateCluster{
		ApplyPolicyDefaultValues:   config.ApplyPolicyDefaultValues,
		Autoscale:                  config.Autoscale,
		AutoterminationMinutes:     config.AutoterminationMinutes,
		AwsAttributes:              config.AwsAttributes,
		AzureAttributes:            config.AzureAttributes,
		ClusterLogConf:             config.ClusterLogConf,
		ClusterName:                config.ClusterName,
		CloneFrom:                  nil, // Not supported by DABs
		CustomTags:                 config.CustomTags,
		DataSecurityMode:           config.DataSecurityMode,
		DependencyMode:             config.DependencyMode,
		DockerImage:                config.DockerImage,
		DriverInstancePoolId:       config.DriverInstancePoolId,
		DriverNodeTypeId:           config.DriverNodeTypeId,
		DriverNodeTypeFlexibility:  config.DriverNodeTypeFlexibility,
		EnableElasticDisk:          config.EnableElasticDisk,
		EnableLocalDiskEncryption:  config.EnableLocalDiskEncryption,
		GcpAttributes:              config.GcpAttributes,
		InitScripts:                config.InitScripts,
		InstancePoolId:             config.InstancePoolId,
		IsSingleNode:               config.IsSingleNode,
		Kind:                       config.Kind,
		NodeTypeId:                 config.NodeTypeId,
		NumWorkers:                 config.NumWorkers,
		PolicyId:                   config.PolicyId,
		RemoteDiskThroughput:       config.RemoteDiskThroughput,
		RuntimeEngine:              config.RuntimeEngine,
		SingleUserName:             config.SingleUserName,
		SparkConf:                  config.SparkConf,
		SparkEnvVars:               config.SparkEnvVars,
		SparkVersion:               config.SparkVersion,
		SshPublicKeys:              config.SshPublicKeys,
		TotalInitialRemoteDiskSize: config.TotalInitialRemoteDiskSize,
		UseMlRuntime:               config.UseMlRuntime,
		WorkloadType:               config.WorkloadType,
		WorkerNodeTypeFlexibility:  config.WorkerNodeTypeFlexibility,
		ForceSendFields:            utils.FilterFields[compute.CreateCluster](config.ForceSendFields),
	}

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	// If NumWorkers is not nil, we don't need to set it to ForceSendFields as it will be sent anyway.
	if config.Autoscale == nil && config.NumWorkers == 0 {
		create.ForceSendFields = append(create.ForceSendFields, "NumWorkers")
	}

	return create
}

func makeEditCluster(id string, config *compute.ClusterSpec) compute.EditCluster {
	edit := compute.EditCluster{
		ClusterId:                  id,
		ApplyPolicyDefaultValues:   config.ApplyPolicyDefaultValues,
		Autoscale:                  config.Autoscale,
		AutoterminationMinutes:     config.AutoterminationMinutes,
		AwsAttributes:              config.AwsAttributes,
		AzureAttributes:            config.AzureAttributes,
		ClusterLogConf:             config.ClusterLogConf,
		ClusterName:                config.ClusterName,
		CustomTags:                 config.CustomTags,
		DataSecurityMode:           config.DataSecurityMode,
		DependencyMode:             config.DependencyMode,
		DockerImage:                config.DockerImage,
		DriverInstancePoolId:       config.DriverInstancePoolId,
		DriverNodeTypeId:           config.DriverNodeTypeId,
		DriverNodeTypeFlexibility:  config.DriverNodeTypeFlexibility,
		EnableElasticDisk:          config.EnableElasticDisk,
		EnableLocalDiskEncryption:  config.EnableLocalDiskEncryption,
		GcpAttributes:              config.GcpAttributes,
		InitScripts:                config.InitScripts,
		InstancePoolId:             config.InstancePoolId,
		IsSingleNode:               config.IsSingleNode,
		Kind:                       config.Kind,
		NodeTypeId:                 config.NodeTypeId,
		NumWorkers:                 config.NumWorkers,
		PolicyId:                   config.PolicyId,
		RemoteDiskThroughput:       config.RemoteDiskThroughput,
		RuntimeEngine:              config.RuntimeEngine,
		SingleUserName:             config.SingleUserName,
		SparkConf:                  config.SparkConf,
		SparkEnvVars:               config.SparkEnvVars,
		SparkVersion:               config.SparkVersion,
		SshPublicKeys:              config.SshPublicKeys,
		TotalInitialRemoteDiskSize: config.TotalInitialRemoteDiskSize,
		UseMlRuntime:               config.UseMlRuntime,
		WorkloadType:               config.WorkloadType,
		WorkerNodeTypeFlexibility:  config.WorkerNodeTypeFlexibility,
		ForceSendFields:            utils.FilterFields[compute.EditCluster](config.ForceSendFields),
	}

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	// If NumWorkers is not nil, we don't need to set it to ForceSendFields as it will be sent anyway.
	if config.Autoscale == nil && config.NumWorkers == 0 {
		edit.ForceSendFields = append(edit.ForceSendFields, "NumWorkers")
	}

	return edit
}

// librariesPath is the ClusterState path of the libraries slice, used to detect library changes.
var librariesPath = structpath.MustParsePath("libraries")

// KeyedSlices compares libraries by identity rather than index so reordering (or the
// read-order the status API returns) does not produce phantom diffs.
func (*ResourceCluster) KeyedSlices() map[string]any {
	return map[string]any{"libraries": libraryKey}
}

// reconcileLibraries uninstalls libraries dropped from config and installs the desired set.
// The Libraries API exposes install and uninstall as separate endpoints, so this is two calls.
func (r *ResourceCluster) reconcileLibraries(ctx context.Context, id string, desired []compute.Library, entry *PlanEntry) error {
	removed := removedLibraries(desired, entry)
	if len(removed) > 0 {
		err := r.client.Libraries.Uninstall(ctx, compute.UninstallLibraries{ClusterId: id, Libraries: removed})
		if err != nil {
			return err
		}
	}
	if len(desired) > 0 {
		return r.client.Libraries.Install(ctx, compute.InstallLibraries{ClusterId: id, Libraries: desired})
	}
	return nil
}

// removedLibraries returns libraries present in the remote state but absent from the desired set.
func removedLibraries(desired []compute.Library, entry *PlanEntry) []compute.Library {
	remote, ok := entry.RemoteState.(*ClusterRemote)
	if !ok || remote == nil {
		return nil
	}
	desiredKeys := make(map[string]struct{}, len(desired))
	for _, l := range desired {
		desiredKeys[libraryMapKey(l)] = struct{}{}
	}
	var result []compute.Library
	for _, l := range remote.Libraries {
		if _, ok := desiredKeys[libraryMapKey(l)]; !ok {
			result = append(result, l)
		}
	}
	return result
}

// restartIfRunning restarts the cluster so a library change takes effect, but only when it is
// running: a stopped cluster applies pending install/uninstall on its next start. It waits for
// the cluster to return to RUNNING before returning.
func (r *ResourceCluster) restartIfRunning(ctx context.Context, id string) error {
	details, err := r.client.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		return err
	}
	if details.State != compute.StateRunning {
		log.Debugf(ctx, "cluster %s is not running (%s); skipping restart for library change", id, details.State)
		return nil
	}
	cmdio.LogString(ctx, fmt.Sprintf("Restarting cluster %s because its libraries changed", id))
	wait, err := r.client.Clusters.Restart(ctx, compute.RestartCluster{ClusterId: id, RestartUser: "", ForceSendFields: nil})
	if err != nil {
		return err
	}
	_, err = wait.GetWithTimeout(clusterWaitTimeout)
	return err
}

// waitForInstall polls until every desired library reaches a terminal installed state. It returns
// early without waiting when the cluster is not running: installs only progress on a running
// cluster and are queued until it next starts.
func (r *ResourceCluster) waitForInstall(ctx context.Context, id string, desired []compute.Library) error {
	if len(desired) == 0 {
		return nil
	}
	details, err := r.client.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		return err
	}
	if details.State != compute.StateRunning {
		log.Debugf(ctx, "cluster %s is not running (%s); skipping wait for library installation", id, details.State)
		return nil
	}

	desiredKeys := make(map[string]struct{}, len(desired))
	for _, l := range desired {
		typ, val := libraryWaitKey(l)
		if typ == "" {
			// Unknown library type: it can't be matched in cluster-status, so don't wait for it.
			continue
		}
		desiredKeys[typ+"="+val] = struct{}{}
	}

	_, err = retries.Poll(ctx, librariesWaitTimeout, func() (*struct{}, *retries.Err) {
		statuses, err := r.client.Libraries.ClusterStatusByClusterId(ctx, id)
		if err != nil {
			return nil, retries.Halt(err)
		}
		pending := len(desiredKeys)
		for _, s := range statuses.LibraryStatuses {
			if s.Library == nil {
				continue
			}
			if _, ok := desiredKeys[libraryWaitMapKey(*s.Library)]; !ok {
				continue
			}
			switch s.Status {
			case compute.LibraryInstallStatusFailed:
				return nil, retries.Halt(fmt.Errorf("library %s failed to install: %s", libraryWaitMapKey(*s.Library), strings.Join(s.Messages, "; ")))
			case compute.LibraryInstallStatusInstalled, compute.LibraryInstallStatusSkipped, compute.LibraryInstallStatusRestored:
				pending--
			case compute.LibraryInstallStatusPending, compute.LibraryInstallStatusResolving, compute.LibraryInstallStatusInstalling, compute.LibraryInstallStatusUninstallOnRestart:
				// Still in progress (or being removed); keep polling.
			}
		}
		if pending > 0 {
			return nil, retries.Continues(fmt.Sprintf("waiting for %d librar(ies) to install on cluster %s", pending, id))
		}
		return &struct{}{}, nil
	})
	return err
}

// libraryWaitKey identifies a library by its primary field only, used to match install-status reports.
func libraryWaitKey(l compute.Library) (string, string) {
	switch {
	case l.Whl != "":
		return "whl", l.Whl
	case l.Jar != "":
		return "jar", l.Jar
	case l.Egg != "":
		return "egg", l.Egg
	case l.Requirements != "":
		return "requirements", l.Requirements
	case l.Pypi != nil:
		return "pypi", l.Pypi.Package
	case l.Maven != nil:
		return "maven", l.Maven.Coordinates
	case l.Cran != nil:
		return "cran", l.Cran.Package
	}
	return "", ""
}

// libraryKey extends libraryWaitKey with repo/exclusions so a repo-only change is detected.
func libraryKey(l compute.Library) (string, string) {
	typ, id := libraryWaitKey(l)
	switch {
	case l.Pypi != nil:
		return typ, id + ";" + l.Pypi.Repo
	case l.Maven != nil:
		return typ, id + ";" + l.Maven.Repo + ";" + strings.Join(l.Maven.Exclusions, ",")
	case l.Cran != nil:
		return typ, id + ";" + l.Cran.Repo
	}
	return typ, id
}

// libraryMapKey flattens libraryKey into a single string for map lookups.
func libraryMapKey(l compute.Library) string {
	f, v := libraryKey(l)
	return f + "=" + v
}

// libraryWaitMapKey flattens libraryWaitKey into a single string for map lookups.
func libraryWaitMapKey(l compute.Library) string {
	f, v := libraryWaitKey(l)
	return f + "=" + v
}
