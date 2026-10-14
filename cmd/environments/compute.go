package environments

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/listing"
	localenv "github.com/databricks/cli/libs/localenv"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/utils"
	compute "github.com/databricks/sdk-go/clusters/v2"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
)

// sdkCompute adapts the Databricks SDK to the localenv.ComputeClient interface.
type sdkCompute struct {
	w *sdkclients.Clients
}

// GetClusterSparkVersion returns the Spark version string for a running cluster.
func (c sdkCompute) GetClusterSparkVersion(ctx context.Context, clusterID string) (string, error) {
	d, err := c.w.Clusters.GetCluster(ctx, compute.GetClusterRequest{ClusterId: new(clusterID)})
	if err != nil {
		return "", fmt.Errorf("get cluster %s: %w", clusterID, err)
	}
	return utils.Deref(d.SparkVersion), nil
}

// GetClusterByName resolves a cluster name to its ID and Spark version.
//
// It intentionally does not use the SDK's GetByClusterName, which lists clusters
// in every state and errors on any name collision: a terminated cluster sharing
// a name with a live one would then block resolution for a name the user
// reasonably considers unique. Instead we list only non-terminated clusters and
// match by name. A name that is still ambiguous among live clusters, or matches
// none, is a genuine error surfaced to the caller as an actionable E_RESOLVE.
//
// The clusters/list API omits clusters terminated more than 30 days ago, so a
// name that only ever belonged to such a cluster resolves as "no active cluster
// named"; this is acceptable since a long-terminated cluster is not a usable
// target and the user can always pass --cluster-id.
func (c sdkCompute) GetClusterByName(ctx context.Context, name string) (string, string, error) {
	clusters, err := listing.Collect(c.w.Clusters.ListClustersIter(ctx, compute.ListClustersRequest{}))
	if err != nil {
		return "", "", fmt.Errorf("list clusters to resolve name %q: %w", name, err)
	}
	var matches []compute.ClusterInfo
	for _, cl := range clusters {
		// TERMINATING is excluded alongside TERMINATED: a cluster on its way
		// down is not a usable target, and keeping it would reintroduce the
		// stale-collision problem this filter exists to avoid.
		switch cl.State {
		case compute.ClusterState_ClusterState_Pending,
			compute.ClusterState_ClusterState_Running,
			compute.ClusterState_ClusterState_Restarting,
			compute.ClusterState_ClusterState_Resizing:
		default:
			continue
		}
		if utils.Deref(cl.ClusterName) == name {
			matches = append(matches, cl)
		}
	}
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("no active cluster named %q", name)
	case 1:
		return utils.Deref(matches[0].ClusterId), utils.Deref(matches[0].SparkVersion), nil
	default:
		return "", "", fmt.Errorf("there are %d active clusters named %q; use --cluster-id to disambiguate", len(matches), name)
	}
}

// JobTaskEnvironment resolves a single job task's compute to an environment.
//
// A job can bind multiple tasks to different serverless environment versions, so
// the target is a specific task, not the whole job. taskKey selects it:
//   - an empty taskKey is an enumerate request — it returns *localenv.ErrTaskKeyRequired
//     carrying the job's task keys, so the caller emits an actionable E_USAGE;
//   - an unknown taskKey returns an error listing the available keys.
//
// A serverless task binds an environment_key to one of the job's environments;
// its version is read directly from that environment's spec (no fallback). A
// classic task resolves from its new_cluster, its job_cluster_key (a shared
// job_clusters entry), or its existing_cluster_id (via the Clusters API). A
// for_each_task is unwrapped to its nested task, whose compute is resolved the
// same way.
func (c sdkCompute) JobTaskEnvironment(ctx context.Context, jobID, taskKey string) (sparkVersion string, isServerless bool, version string, err error) {
	id, err := strconv.ParseInt(jobID, 10, 64)
	if err != nil {
		return "", false, "", fmt.Errorf("invalid job ID %q: must be an integer: %w", jobID, err)
	}

	sdkJob, err := c.w.Jobs.GetJob(ctx, sdkjobs.GetJobRequest{JobId: new(id)})
	if err != nil {
		return "", false, "", fmt.Errorf("get job %d: %w", id, err)
	}
	job, err := jobs.GetJobResponseFromSDK(sdkJob)
	if err != nil {
		return "", false, "", fmt.Errorf("get job %d: %w", id, err)
	}
	if job.Settings == nil {
		return "", false, "", fmt.Errorf("job %d has no settings", id)
	}

	taskKeys := make([]string, 0, len(job.Settings.Tasks))
	for _, t := range job.Settings.Tasks {
		taskKeys = append(taskKeys, utils.Deref(t.TaskKey))
	}

	// No task key given: ask the caller to pick one, listing what is available.
	// This is a usage error, not a resolve failure — see ResolveCompute.
	if taskKey == "" {
		return "", false, "", &localenv.ErrTaskKeyRequired{JobID: jobID, TaskKeys: taskKeys}
	}

	var task *jobs.TaskSettings
	for i := range job.Settings.Tasks {
		if utils.Deref(job.Settings.Tasks[i].TaskKey) == taskKey {
			task = &job.Settings.Tasks[i]
			break
		}
	}
	if task == nil {
		return "", false, "", fmt.Errorf("job %s has no task %q (available: %s)", jobID, taskKey, strings.Join(taskKeys, ", "))
	}

	// A for_each_task wraps the real per-iteration task: its compute
	// (environment_key / new_cluster / existing_cluster_id / job_cluster_key)
	// lives on the nested task, not the outer one. Resolve against that.
	if task.ForEachTask != nil {
		task = task.ForEachTask.Task
	}

	return c.resolveTaskCompute(ctx, job.Settings, task, jobID, taskKey)
}

// resolveTaskCompute resolves one task's compute to an environment. It reads the
// serverless environment version directly (no fallback) or the classic cluster's
// Spark version, consulting the job-level environments and job_clusters the task
// references by key.
func (c sdkCompute) resolveTaskCompute(ctx context.Context, settings *jobs.JobSettings, task *jobs.TaskSettings, jobID, taskKey string) (sparkVersion string, isServerless bool, version string, err error) {
	// Serverless task: it references one of the job's environments by key; the
	// version lives on that environment's spec and is used directly.
	if utils.Deref(task.EnvironmentKey) != "" {
		for _, e := range settings.Environments {
			if e.EnvironmentKey == task.EnvironmentKey {
				v := environmentVersion(e)
				if v == "" {
					return "", false, "", fmt.Errorf("task %q of job %s binds environment %q, which records no environment version", taskKey, jobID, utils.Deref(task.EnvironmentKey))
				}
				return "", true, v, nil
			}
		}
		return "", false, "", fmt.Errorf("task %q of job %s references environment %q, which the job does not define", taskKey, jobID, utils.Deref(task.EnvironmentKey))
	}

	// Classic task with an inline cluster spec.
	if task.NewCluster != nil && utils.Deref(task.NewCluster.SparkVersion) != "" {
		return utils.Deref(task.NewCluster.SparkVersion), false, utils.Deref(task.NewCluster.SparkVersion), nil
	}

	// Classic task referencing a shared job cluster by key: read that cluster's
	// Spark version from the job's job_clusters (the common classic-task shape).
	if utils.Deref(task.JobClusterKey) != "" {
		for _, jc := range settings.JobClusters {
			if jc.JobClusterKey == task.JobClusterKey {
				if jc.NewCluster == nil || utils.Deref(jc.NewCluster.SparkVersion) == "" {
					return "", false, "", fmt.Errorf("task %q of job %s uses job cluster %q, which has no spark_version", taskKey, jobID, utils.Deref(task.JobClusterKey))
				}
				return utils.Deref(jc.NewCluster.SparkVersion), false, utils.Deref(jc.NewCluster.SparkVersion), nil
			}
		}
		return "", false, "", fmt.Errorf("task %q of job %s references job cluster %q, which the job does not define", taskKey, jobID, utils.Deref(task.JobClusterKey))
	}

	// Classic task pinned to an existing cluster: resolve its Spark version.
	if utils.Deref(task.ExistingClusterId) != "" {
		sv, cerr := c.GetClusterSparkVersion(ctx, utils.Deref(task.ExistingClusterId))
		if cerr != nil {
			return "", false, "", fmt.Errorf("resolving existing cluster %s for task %q of job %s: %w", utils.Deref(task.ExistingClusterId), taskKey, jobID, cerr)
		}
		return sv, false, sv, nil
	}

	return "", false, "", fmt.Errorf("task %q of job %s has no serverless environment or resolvable cluster; pass --cluster-id or --serverless-version explicitly", taskKey, jobID)
}

// environmentVersion returns the serverless environment version recorded on a
// job environment, or "" when the spec or version is absent.
//
// The version can arrive in either of two fields. environment_version is the
// current one; client is its deprecated predecessor ("Use environment_version
// instead") and is still what some jobs pin. Reading both means a client-pinned
// task is not treated as unversioned. base_environment is deliberately ignored:
// it is a path/ID, not a version.
func environmentVersion(e jobs.JobEnvironment) string {
	if e.Spec == nil {
		return ""
	}
	if utils.Deref(e.Spec.EnvironmentVersion) != "" {
		return utils.Deref(e.Spec.EnvironmentVersion)
	}
	return utils.Deref(e.Spec.Client)
}
