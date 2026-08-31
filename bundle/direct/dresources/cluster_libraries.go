package dresources

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
)

// Corresponds to the databricks_library terraform resource:
// https://registry.terraform.io/providers/databricks/databricks/latest/docs/resources/library

// librariesWaitTimeout bounds how long we poll for libraries to finish installing.
const librariesWaitTimeout = 15 * time.Minute

// LibrariesState is the state for a cluster's libraries sub-resource. Libraries are installed
// via the Libraries API against the parent cluster identified by ClusterId, not through the
// cluster spec.
type LibrariesState struct {
	ClusterId string `json:"cluster_id"`
	// By convention EmbeddedSlice fields have the __embed__ json tag, see permissions.go.
	EmbeddedSlice []compute.Library `json:"__embed__,omitempty"`
}

type ResourceLibraries struct {
	client *databricks.WorkspaceClient
}

func (*ResourceLibraries) New(client *databricks.WorkspaceClient) *ResourceLibraries {
	return &ResourceLibraries{client: client}
}

func (r *ResourceLibraries) PrepareInputConfig(inputConfig *[]compute.Library, resourceKey string) (*structvar.StructVar, error) {
	baseNode, ok := strings.CutSuffix(resourceKey, ".libraries")
	if !ok {
		return nil, fmt.Errorf("internal error: node %q does not end with .libraries", resourceKey)
	}

	return &structvar.StructVar{
		Value: &LibrariesState{
			ClusterId:     "", // Always a reference, defined in Refs below.
			EmbeddedSlice: *inputConfig,
		},
		Refs: map[string]string{
			"cluster_id": "${" + baseNode + ".id}",
		},
	}, nil
}

func (*ResourceLibraries) PrepareState(state *LibrariesState) *LibrariesState {
	return state
}

// IsEmptyState reports an empty libraries list as no resource at all: nothing to install, and no
// state entry is persisted for it.
func (*ResourceLibraries) IsEmptyState(state *LibrariesState) bool {
	return len(state.EmbeddedSlice) == 0
}

// libraryKey identifies a library by its type-specific field so slices compare by identity
// rather than by index (see KeyedSlices).
func libraryKey(l compute.Library) (string, string) {
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

func (*ResourceLibraries) KeyedSlices() map[string]any {
	// Empty key because EmbeddedSlice appears at the root path of LibrariesState.
	return map[string]any{
		"": libraryKey,
	}
}

func (r *ResourceLibraries) DoRead(ctx context.Context, id string) (*LibrariesState, error) {
	statuses, err := r.client.Libraries.ClusterStatusByClusterId(ctx, id)
	if err != nil {
		return nil, err
	}

	state := &LibrariesState{ClusterId: id, EmbeddedSlice: nil}
	for _, s := range statuses.LibraryStatuses {
		// Libraries set for all clusters via the UI are not managed by the bundle
		// (following the permissions convention of ignoring inherited entries).
		if s.Library == nil || s.IsLibraryForAllClusters {
			continue
		}
		// A library pending uninstall on restart is on its way out; don't report it as present.
		if s.Status == compute.LibraryInstallStatusUninstallOnRestart {
			continue
		}
		state.EmbeddedSlice = append(state.EmbeddedSlice, *s.Library)
	}
	return state, nil
}

// DoCreate installs the libraries on the cluster.
// https://docs.databricks.com/api/workspace/libraries/install
func (r *ResourceLibraries) DoCreate(ctx context.Context, state *LibrariesState) (string, *LibrariesState, error) {
	err := r.client.Libraries.Install(ctx, compute.InstallLibraries{
		ClusterId: state.ClusterId,
		Libraries: state.EmbeddedSlice,
	})
	if err != nil {
		// Install is idempotent (installing an already-installed library is a no-op),
		// so retrying on transient errors is safe.
		return "", nil, retrySafe(err)
	}
	return state.ClusterId, nil, nil
}

// DoUpdate uninstalls libraries removed from config and installs the desired set. This is two API
// calls because the Libraries API exposes install and uninstall as separate endpoints, unlike the
// single-call model most resources follow.
func (r *ResourceLibraries) DoUpdate(ctx context.Context, id string, state *LibrariesState, entry *PlanEntry) (*LibrariesState, error) {
	removed := removedLibraries(state.EmbeddedSlice, entry)
	if len(removed) > 0 {
		err := r.client.Libraries.Uninstall(ctx, compute.UninstallLibraries{
			ClusterId: id,
			Libraries: removed,
		})
		if err != nil {
			return nil, err
		}
	}

	if len(state.EmbeddedSlice) > 0 {
		err := r.client.Libraries.Install(ctx, compute.InstallLibraries{
			ClusterId: id,
			Libraries: state.EmbeddedSlice,
		})
		if err != nil {
			return nil, err
		}
	}
	return nil, nil
}

// DoDelete uninstalls the libraries recorded in state. It fires both when the whole libraries
// block is removed from config (the cluster is kept) and during bundle destroy (the cluster is
// deleted too). WaitAfterDelete then restarts the cluster so the uninstall takes effect, since the
// Libraries API defers uninstalls until the next restart.
//
// TODO: during `bundle destroy` this restart is wasteful, since the cluster is permanently deleted
// right after. The delete path only receives the cluster id, so it cannot tell "block removed,
// cluster kept" from "cluster destroyed" and the restart fires in both. Worth revisiting if the
// framework can signal that the parent is also being deleted.
func (r *ResourceLibraries) DoDelete(ctx context.Context, id string, state *LibrariesState) error {
	if len(state.EmbeddedSlice) == 0 {
		return nil
	}
	// A 404 here means the parent cluster is already gone; the framework treats that as success.
	return r.client.Libraries.Uninstall(ctx, compute.UninstallLibraries{
		ClusterId: id,
		Libraries: state.EmbeddedSlice,
	})
}

// removedLibraries returns libraries present in the remote state but absent from the desired set.
func removedLibraries(desired []compute.Library, entry *PlanEntry) []compute.Library {
	if entry == nil {
		return nil
	}
	remote, ok := entry.RemoteState.(*LibrariesState)
	if !ok || remote == nil {
		return nil
	}

	desiredKeys := make(map[string]struct{}, len(desired))
	for _, l := range desired {
		desiredKeys[libraryMapKey(l)] = struct{}{}
	}

	var result []compute.Library
	for _, l := range remote.EmbeddedSlice {
		if _, ok := desiredKeys[libraryMapKey(l)]; !ok {
			result = append(result, l)
		}
	}
	return result
}

// libraryMapKey flattens libraryKey into a single string for map lookups.
func libraryMapKey(l compute.Library) string {
	f, v := libraryKey(l)
	return f + "=" + v
}

func (r *ResourceLibraries) WaitAfterCreate(ctx context.Context, id string, state *LibrariesState) (*LibrariesState, error) {
	return nil, r.waitForInstall(ctx, id, state.EmbeddedSlice)
}

func (r *ResourceLibraries) WaitAfterUpdate(ctx context.Context, id string, state *LibrariesState) (*LibrariesState, error) {
	// Restart so the change is live: newly installed libraries are invisible to attached notebooks
	// until restart, and the Libraries API defers uninstalls to the next restart.
	if err := r.restartIfRunning(ctx, id); err != nil {
		return nil, err
	}
	return nil, r.waitForInstall(ctx, id, state.EmbeddedSlice)
}

// WaitAfterDelete restarts the cluster so libraries uninstalled by DoDelete are actually evicted
// (the Libraries API defers uninstalls until restart).
func (r *ResourceLibraries) WaitAfterDelete(ctx context.Context, id string) error {
	err := r.restartIfRunning(ctx, id)
	if apierr.IsMissing(err) {
		// Parent cluster already deleted; nothing to restart.
		return nil
	}
	return err
}

// restartIfRunning restarts the cluster so library changes take effect, but only when it is
// running: a stopped cluster applies pending install/uninstall on its next start. It waits for the
// cluster to return to RUNNING before returning.
func (r *ResourceLibraries) restartIfRunning(ctx context.Context, id string) error {
	details, err := r.client.Clusters.GetByClusterId(ctx, id)
	if err != nil {
		return err
	}
	if details.State != compute.StateRunning {
		log.Debugf(ctx, "cluster %s is not running (%s); skipping restart for library change", id, details.State)
		return nil
	}

	cmdio.LogString(ctx, fmt.Sprintf("Restarting cluster %s because its libraries changed", id))
	wait, err := r.client.Clusters.Restart(ctx, compute.RestartCluster{
		ClusterId:       id,
		RestartUser:     "",
		ForceSendFields: nil,
	})
	if err != nil {
		return err
	}
	_, err = wait.GetWithTimeout(clusterWaitTimeout)
	return err
}

// waitForInstall polls until every desired library reaches a terminal installed state. It returns
// early without waiting when the cluster is not running: installs only progress on a running
// cluster and are queued until it next starts.
func (r *ResourceLibraries) waitForInstall(ctx context.Context, id string, desired []compute.Library) error {
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
		desiredKeys[libraryMapKey(l)] = struct{}{}
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
			if _, ok := desiredKeys[libraryMapKey(*s.Library)]; !ok {
				continue
			}
			switch s.Status {
			case compute.LibraryInstallStatusFailed:
				return nil, retries.Halt(fmt.Errorf("library %s failed to install: %s", libraryMapKey(*s.Library), strings.Join(s.Messages, "; ")))
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
