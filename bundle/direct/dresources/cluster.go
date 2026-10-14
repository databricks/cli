package dresources

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/internal/retries"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/clusterlibraries/v2"
	"github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/sdktypes/sdkconv"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/utils"
	sdkclusterlibraries "github.com/databricks/sdk-go/clusterlibraries/v2"
	sdkclusters "github.com/databricks/sdk-go/clusters/v2"
	"golang.org/x/sync/errgroup"
)

// librariesWaitTimeout bounds how long we poll for libraries to finish installing.
const librariesWaitTimeout = 15 * time.Minute

// clusterWaitTimeout bounds how long we poll for a cluster to reach its target
// state (RUNNING/TERMINATED) after create, edit, or start/stop. Provisioning can
// legitimately take longer than 15 minutes on capacity-constrained workspaces
// (a cluster stays PENDING with "Finding instances for new nodes"), so we allow
// 30 minutes before giving up. Terminal states still halt immediately.
const clusterWaitTimeout = 30 * time.Minute

// ClusterState is the state type for Cluster resources. It extends clusters.CreateClusterRequest with
// lifecycle settings and the cluster ID.
// ClusterId is written to state by DoCreate/DoUpdate for informational purposes; it is not
// used in diff computation (neither PrepareState nor RemapState set it).
type ClusterState struct {
	clusters.CreateClusterRequest

	Lifecycle *StateLifecycle `json:"lifecycle,omitempty"`

	// Libraries are installed via the Libraries API, not the cluster spec, and managed as
	// part of the cluster (see reconcileLibraries and the install in WaitAfterCreate).
	Libraries []clusterlibraries.Library `json:"libraries,omitempty"`
}

// ClusterRemote extends clusters.ClusterInfo with a synthetic Lifecycle field so that
// RemoteType satisfies TestRemoteSuperset (every field in ClusterState exists in ClusterRemote).
// Lifecycle.Started is populated by DoRead from the cluster's running state.
//
// ApplyPolicyDefaultValues is promoted to the top level because the cluster GET API returns it
// only under .spec (a snapshot of the create/edit settings), never at the top level. DoRead
// copies it up so RemapState stays a dumb copy and the field participates in normal drift
// detection instead of being suppressed as missing_in_remote.
type ClusterRemote struct {
	clusters.ClusterInfo
	ApplyPolicyDefaultValues *bool           `json:"apply_policy_default_values,omitempty"`
	Lifecycle                *StateLifecycle `json:"lifecycle,omitempty"`

	// Libraries is populated by DoRead from the Libraries cluster-status API (the cluster
	// GET does not return installed libraries), so it participates in drift detection.
	Libraries []clusterlibraries.Library `json:"libraries,omitempty"`
}

type ResourceCluster struct {
	client *sdkclients.Clients
}

func (r *ResourceCluster) New(client *sdkclients.Clients) any {
	return &ResourceCluster{
		client: client,
	}
}

func (r *ResourceCluster) PrepareState(input *resources.Cluster) *ClusterState {
	s := &ClusterState{
		CreateClusterRequest: input.CreateClusterRequest,
		Lifecycle:            nil,
		Libraries:            input.Libraries,
	}
	if input.Lifecycle != nil && input.Lifecycle.Started != nil {
		s.Lifecycle = &StateLifecycle{Started: input.Lifecycle.Started}
	}
	return s
}

// RemapState maps the remote ClusterRemote to ClusterState for diff comparison.
// Started is derived from cluster state so the planner can detect start/stop changes.
func (r *ResourceCluster) RemapState(input *ClusterRemote) *ClusterState {
	spec := &ClusterState{
		CreateClusterRequest: clusters.CreateClusterRequest{
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
		},
		Lifecycle: input.Lifecycle,
		Libraries: input.Libraries,
	}
	return spec
}

func (r *ResourceCluster) DoRead(ctx context.Context, id string) (*ClusterRemote, error) {
	var details *clusters.ClusterInfo
	var libraries []clusterlibraries.Library

	// The cluster GET and the library-status GET are independent, so run them concurrently.
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		details, err = r.getCluster(ctx, id)
		return err
	})
	g.Go(func() error {
		var err error
		libraries, err = r.readLibraries(ctx, id)
		return err
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	remote := &ClusterRemote{
		ClusterInfo:              *details,
		ApplyPolicyDefaultValues: nil,
		Lifecycle:                nil,
		Libraries:                libraries,
	}
	// The GET response carries apply_policy_default_values only under .spec (a snapshot of the
	// create/edit settings), not at the top level. Promote it so RemapState is a dumb copy.
	if details.Spec != nil {
		remote.ApplyPolicyDefaultValues = details.Spec.ApplyPolicyDefaultValues
	}

	switch details.State {
	case clusters.ClusterState_ClusterState_Running:
		remote.Lifecycle = &StateLifecycle{Started: new(true)}
	case clusters.ClusterState_ClusterState_Terminated:
		remote.Lifecycle = &StateLifecycle{Started: new(false)}
	default:
		remote.Lifecycle = nil
	}

	return remote, nil
}

func (r *ResourceCluster) getCluster(ctx context.Context, id string) (*clusters.ClusterInfo, error) {
	info, err := r.client.Clusters.GetCluster(ctx, sdkclusters.GetClusterRequest{ClusterId: new(id)})
	if err != nil {
		return nil, err
	}
	return clusters.ClusterInfoFromSDK(info)
}

// waitClusterState polls until the cluster reaches target. Reaching one of the
// failure states first is an error.
func (r *ResourceCluster) waitClusterState(ctx context.Context, id string, target clusters.ClusterState_ClusterState, failure ...clusters.ClusterState_ClusterState) error {
	_, err := retries.Poll(ctx, clusterWaitTimeout, func() (*clusters.ClusterInfo, *retries.Err) {
		details, err := r.getCluster(ctx, id)
		if err != nil {
			return nil, retries.Halt(err)
		}
		status := details.State
		statusMessage := utils.Deref(details.StateMessage)
		if status == target {
			return details, nil
		}
		if slices.Contains(failure, status) {
			return nil, retries.Halt(fmt.Errorf("failed to reach %s, got %s: %s", target, status, statusMessage))
		}
		return nil, retries.Continues(statusMessage)
	})
	return err
}

func (r *ResourceCluster) waitRunning(ctx context.Context, id string) error {
	return r.waitClusterState(ctx, id, clusters.ClusterState_ClusterState_Running, clusters.ClusterState_ClusterState_Error, clusters.ClusterState_ClusterState_Terminated)
}

func (r *ResourceCluster) waitTerminated(ctx context.Context, id string) error {
	return r.waitClusterState(ctx, id, clusters.ClusterState_ClusterState_Terminated, clusters.ClusterState_ClusterState_Error)
}

func (r *ResourceCluster) clusterStatus(ctx context.Context, id string) (*clusterlibraries.ClusterLibraryStatuses, error) {
	resp, err := r.client.Libraries.ClusterStatus(ctx, sdkclusterlibraries.ClusterStatusRequest{ClusterId: new(id)})
	if err != nil {
		return nil, err
	}
	return clusterlibraries.ClusterLibraryStatusesFromSDK(resp)
}

// readLibraries returns the bundle-managed libraries installed on the cluster.
// https://docs.databricks.com/api/workspace/libraries/clusterstatus
func (r *ResourceCluster) readLibraries(ctx context.Context, id string) ([]clusterlibraries.Library, error) {
	statuses, err := r.clusterStatus(ctx, id)
	if err != nil {
		return nil, err
	}
	var libraries []clusterlibraries.Library
	for _, s := range statuses.LibraryStatuses {
		// Libraries set for all clusters via the UI are not managed by the bundle; a library
		// pending uninstall on restart is on its way out. Skip both.
		if s.Library == nil || utils.Deref(s.IsLibraryForAllClusters) || s.Status == clusterlibraries.LibraryInstallStatus_UninstallOnRestart {
			continue
		}
		libraries = append(libraries, *s.Library)
	}
	return libraries, nil
}

func (r *ResourceCluster) DoCreate(ctx context.Context, config *ClusterState) (string, *ClusterRemote, error) {
	request, err := makeCreateCluster(&config.CreateClusterRequest)
	if err != nil {
		return "", nil, err
	}
	wait, err := r.client.Clusters.CreateCluster(ctx, request)
	if err != nil {
		return "", nil, err
	}
	return wait.GetClusterId(), nil, nil
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
	if hasClusterSpecChanges(entry) {
		// Same retry as in TF provider logic
		// https://github.com/databricks/terraform-provider-databricks/blob/3eecd0f90cf99d7777e79a3d03c41f9b2aafb004/clusters/resource_cluster.go#L624
		request, err := makeEditCluster(id, &config.CreateClusterRequest)
		if err != nil {
			return nil, err
		}
		_, err = retries.Poll(ctx, clusterWaitTimeout, func() (*sdkclusters.EditClusterWaiter, *retries.Err) {
			wait, err := r.client.Clusters.EditCluster(ctx, request)
			if err == nil {
				return wait, nil
			}

			// Only Running and Terminated clusters can be modified. In particular, autoscaling clusters cannot be modified
			// while the resizing is ongoing. We retry in this case. Scaling can take several minutes.
			if apierrs.ErrorCode(err) == "INVALID_STATE" {
				return nil, retries.Continues(fmt.Sprintf("cluster %s cannot be modified in its current state: %s", id, apierrs.Message(err)))
			}
			return nil, retries.Halt(err)
		})
		if err != nil {
			return nil, err
		}
	}

	// TODO: a local whl/jar whose workspace path is unchanged but whose contents
	// changed (same name+version, non-dev mode) is not detected here, so it is not reinstalled.
	// Dev mode handles this via patchwheel (a source-derived version bump).
	if entry.Changes.HasChange(librariesPath) {
		if err := r.reconcileLibraries(ctx, id, config.Libraries, entry); err != nil {
			return nil, err
		}
		// Installs apply live on a running cluster. Uninstalls are marked UNINSTALL_ON_RESTART
		// by the backend and take effect at the cluster's next restart; the bundle never restarts it.
		if err := r.waitForInstall(ctx, id, config.Libraries); err != nil {
			return nil, err
		}
	}

	if config.Lifecycle == nil || config.Lifecycle.Started == nil {
		return nil, nil
	}

	desiredStarted := *config.Lifecycle.Started
	alreadyRunning := remoteClusterIsRunning(entry)
	if desiredStarted && !alreadyRunning {
		// lifecycle.started=true: fire Start; WaitAfterUpdate polls for RUNNING.
		_, err := r.client.Clusters.StartCluster(ctx, sdkclusters.StartClusterRequest{ClusterId: new(id)})
		return nil, err
	} else if !desiredStarted && alreadyRunning {
		// lifecycle.started=false: fire Delete; WaitAfterUpdate polls for TERMINATED.
		// Note: Delete terminates the cluster; permanent removal is a separate API (permanent-delete).
		_, err := r.client.Clusters.DeleteCluster(ctx, sdkclusters.DeleteClusterRequest{ClusterId: new(id)})
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
		return nil, r.waitRunning(ctx, id)
	}
	return nil, r.waitTerminated(ctx, id)
}

// WaitAfterCreate waits for the cluster to reach RUNNING state (clusters always start on creation).
// When lifecycle.started=false, it then terminates the cluster.
func (r *ResourceCluster) WaitAfterCreate(ctx context.Context, id string, config *ClusterState) (*ClusterRemote, error) {
	// Always wait for RUNNING first: clusters start in PENDING state and must be polled.
	err := r.waitRunning(ctx, id)
	if err != nil {
		return nil, err
	}

	// Install libraries once the cluster is running: installs only progress on a running cluster.
	// TODO: Wait is supposed to be side effect free, but in this case moving it to
	// the create will cause a wait for libraries to be installed befor the cluster is installed.
	// this increases the risk of losing the cluster. This is a limitation
	if len(config.Libraries) > 0 {
		err = r.installLibraries(ctx, id, config.Libraries)
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
		_, err := r.client.Clusters.DeleteCluster(ctx, sdkclusters.DeleteClusterRequest{ClusterId: new(id)})
		if err != nil {
			return nil, err
		}
		return nil, r.waitTerminated(ctx, id)
	}

	return nil, nil
}

func (r *ResourceCluster) DoResize(ctx context.Context, id string, config *ClusterState, entry *PlanEntry) error {
	request, err := clusters.ResizeClusterRequestToSDK(&clusters.ResizeClusterRequest{
		ClusterId:  new(id),
		NumWorkers: config.NumWorkers,
		Autoscale:  config.Autoscale,
	})
	if err != nil {
		return err
	}
	_, err = r.client.Clusters.ResizeCluster(ctx, *request)
	if err == nil {
		return nil
	}

	if apierrs.ErrorCode(err) != "INVALID_STATE" {
		return err
	}

	// Cluster is not running; fall back to the full clusters/edit path.
	log.Debugf(ctx, "cluster %s: resize returned INVALID_STATE (%s), falling back to edit", id, err)
	_, err = r.DoUpdate(ctx, id, config, entry)
	return err
}

func (r *ResourceCluster) DoDelete(ctx context.Context, id string, _ *ClusterState) error {
	_, err := r.client.Clusters.PermanentDeleteCluster(ctx, sdkclusters.PermanentDeleteClusterRequest{ClusterId: new(id)})
	return err
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
		if change.New == clusters.DataSecurityMode_DataSecurityModeStandard && change.Remote == clusters.DataSecurityMode_UserIsolation && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == clusters.DataSecurityMode_DataSecurityModeDedicated && change.Remote == clusters.DataSecurityMode_SingleUser && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		} else if change.New == clusters.DataSecurityMode_DataSecurityModeAuto && (change.Remote == clusters.DataSecurityMode_SingleUser || change.Remote == clusters.DataSecurityMode_UserIsolation) && change.New == change.Old {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonAlias
		}

	case "num_workers", "autoscale":
		if remoteState != nil && remoteState.State == clusters.ClusterState_ClusterState_Running {
			change.Action = deployplan.Resize
		}

	case "libraries":
		// Without a libraries section the bundle does not manage the cluster's libraries, matching
		// the behaviour before the field existed: job runs install their task libraries cluster-wide,
		// so what the cluster reports is not drift. With no section, New is nil and the difference
		// arrives as a single whole-field change; `libraries: []` keeps New non-nil and stays managed.
		if p.String() == "libraries" && change.New == nil {
			change.Action = deployplan.Skip
			change.Reason = deployplan.ReasonUnmanaged
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
	return remote.State == clusters.ClusterState_ClusterState_Running
}

func makeCreateCluster(config *clusters.CreateClusterRequest) (sdkclusters.CreateClusterRequest, error) {
	create := *config
	create.CloneFrom = nil // Not supported by DABs

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	if create.Autoscale == nil && create.NumWorkers == nil {
		create.NumWorkers = new(0)
	}

	request, err := clusters.CreateClusterRequestToSDK(&create)
	if err != nil {
		return sdkclusters.CreateClusterRequest{}, err
	}
	return *request, nil
}

func makeEditCluster(id string, config *clusters.CreateClusterRequest) (sdkclusters.EditClusterRequest, error) {
	create := *config
	create.CloneFrom = nil // Not supported by the edit API

	// If autoscale is not set, we need to send NumWorkers because one of them is required.
	if create.Autoscale == nil && create.NumWorkers == nil {
		create.NumWorkers = new(0)
	}

	// EditClusterRequest is CreateClusterRequest without clone_from plus the cluster id.
	request, err := sdkconv.ToSDK[sdkclusters.EditClusterRequest](&create)
	if err != nil {
		return sdkclusters.EditClusterRequest{}, err
	}
	request.ClusterId = new(id)
	return *request, nil
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
func (r *ResourceCluster) reconcileLibraries(ctx context.Context, id string, desired []clusterlibraries.Library, entry *PlanEntry) error {
	removed := removedLibraries(desired, entry)
	if len(removed) > 0 {
		request, err := clusterlibraries.UninstallLibrariesRequestToSDK(&clusterlibraries.UninstallLibrariesRequest{ClusterId: new(id), Libraries: removed})
		if err != nil {
			return err
		}
		if _, err := r.client.Libraries.UninstallLibraries(ctx, *request); err != nil {
			return err
		}
	}
	if len(desired) > 0 {
		return r.installLibraries(ctx, id, desired)
	}
	return nil
}

func (r *ResourceCluster) installLibraries(ctx context.Context, id string, libraries []clusterlibraries.Library) error {
	request, err := clusterlibraries.InstallLibrariesRequestToSDK(&clusterlibraries.InstallLibrariesRequest{ClusterId: new(id), Libraries: libraries})
	if err != nil {
		return err
	}
	_, err = r.client.Libraries.InstallLibraries(ctx, *request)
	return err
}

// removedLibraries returns libraries present in the remote state but absent from the desired set.
func removedLibraries(desired []clusterlibraries.Library, entry *PlanEntry) []clusterlibraries.Library {
	remote, ok := entry.RemoteState.(*ClusterRemote)
	if !ok || remote == nil {
		return nil
	}
	desiredKeys := make(map[string]struct{}, len(desired))
	for _, l := range desired {
		desiredKeys[libraryMapKey(l)] = struct{}{}
	}
	var result []clusterlibraries.Library
	for _, l := range remote.Libraries {
		if _, ok := desiredKeys[libraryMapKey(l)]; !ok {
			result = append(result, l)
		}
	}
	return result
}

// waitForInstall polls until every desired library reaches a terminal installed state. It returns
// early without waiting when the cluster is not running: installs only progress on a running
// cluster and are queued until it next starts.
func (r *ResourceCluster) waitForInstall(ctx context.Context, id string, desired []clusterlibraries.Library) error {
	if len(desired) == 0 {
		return nil
	}
	details, err := r.getCluster(ctx, id)
	if err != nil {
		return err
	}
	if details.State != clusters.ClusterState_ClusterState_Running {
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
		statuses, err := r.clusterStatus(ctx, id)
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
			case clusterlibraries.LibraryInstallStatus_Failed:
				return nil, retries.Halt(fmt.Errorf("library %s failed to install: %s", libraryWaitMapKey(*s.Library), strings.Join(s.Messages, "; ")))
			case clusterlibraries.LibraryInstallStatus_Installed, clusterlibraries.LibraryInstallStatus_Skipped, clusterlibraries.LibraryInstallStatus_Restored:
				pending--
			case clusterlibraries.LibraryInstallStatus_Resolving, clusterlibraries.LibraryInstallStatus_Installing, clusterlibraries.LibraryInstallStatus_UninstallOnRestart:
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
func libraryWaitKey(l clusterlibraries.Library) (string, string) {
	switch {
	case utils.Deref(l.Whl) != "":
		return "whl", *l.Whl
	case utils.Deref(l.Jar) != "":
		return "jar", *l.Jar
	case utils.Deref(l.Egg) != "":
		return "egg", *l.Egg
	case utils.Deref(l.Requirements) != "":
		return "requirements", *l.Requirements
	case l.Pypi != nil:
		return "pypi", utils.Deref(l.Pypi.Package)
	case l.Maven != nil:
		return "maven", utils.Deref(l.Maven.Coordinates)
	case l.Cran != nil:
		return "cran", utils.Deref(l.Cran.Package)
	}
	return "", ""
}

// libraryKey extends libraryWaitKey with repo/exclusions so a repo-only change is detected.
func libraryKey(l clusterlibraries.Library) (string, string) {
	typ, id := libraryWaitKey(l)
	switch {
	case l.Pypi != nil:
		return typ, id + ";" + utils.Deref(l.Pypi.Repo)
	case l.Maven != nil:
		return typ, id + ";" + utils.Deref(l.Maven.Repo) + ";" + strings.Join(l.Maven.Exclusions, ",")
	case l.Cran != nil:
		return typ, id + ";" + utils.Deref(l.Cran.Repo)
	}
	return typ, id
}

// libraryMapKey flattens libraryKey into a single string for map lookups.
func libraryMapKey(l clusterlibraries.Library) string {
	f, v := libraryKey(l)
	return f + "=" + v
}

// libraryWaitMapKey flattens libraryWaitKey into a single string for map lookups.
func libraryWaitMapKey(l clusterlibraries.Library) string {
	f, v := libraryWaitKey(l)
	return f + "=" + v
}
