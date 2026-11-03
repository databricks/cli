package environments

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	localenv "github.com/databricks/cli/libs/localenv"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// sdkCompute adapts the Databricks SDK to the localenv.ComputeClient interface.
type sdkCompute struct {
	w *databricks.WorkspaceClient
}

// GetClusterSparkVersion returns the Spark version string for a running cluster.
func (c sdkCompute) GetClusterSparkVersion(ctx context.Context, clusterID string) (string, error) {
	d, err := c.w.Clusters.GetByClusterId(ctx, clusterID)
	if err != nil {
		return "", fmt.Errorf("get cluster %s: %w", clusterID, err)
	}
	return d.SparkVersion, nil
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
	clusters, err := c.w.Clusters.ListAll(ctx, compute.ListClustersRequest{
		FilterBy: &compute.ListClustersFilterBy{
			// TERMINATING is excluded alongside TERMINATED: a cluster on its way
			// down is not a usable target, and keeping it would reintroduce the
			// stale-collision problem this filter exists to avoid.
			ClusterStates: []compute.State{
				compute.StatePending,
				compute.StateRunning,
				compute.StateRestarting,
				compute.StateResizing,
			},
		},
	})
	if err != nil {
		return "", "", fmt.Errorf("list clusters to resolve name %q: %w", name, err)
	}
	var matches []compute.ClusterDetails
	for _, cl := range clusters {
		if cl.ClusterName == name {
			matches = append(matches, cl)
		}
	}
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("no active cluster named %q", name)
	case 1:
		return matches[0].ClusterId, matches[0].SparkVersion, nil
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

	job, err := c.w.Jobs.GetByJobId(ctx, id)
	if err != nil {
		return "", false, "", fmt.Errorf("get job %d: %w", id, err)
	}
	if job.Settings == nil {
		return "", false, "", fmt.Errorf("job %d has no settings", id)
	}

	taskKeys := make([]string, 0, len(job.Settings.Tasks))
	for _, t := range job.Settings.Tasks {
		taskKeys = append(taskKeys, t.TaskKey)
	}

	// No task key given: ask the caller to pick one, listing what is available.
	// This is a usage error, not a resolve failure — see ResolveCompute.
	if taskKey == "" {
		return "", false, "", &localenv.ErrTaskKeyRequired{JobID: jobID, TaskKeys: taskKeys}
	}

	var task *jobs.Task
	for i := range job.Settings.Tasks {
		if job.Settings.Tasks[i].TaskKey == taskKey {
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
		task = &task.ForEachTask.Task
	}

	return c.resolveTaskCompute(ctx, job.Settings, task, jobID, taskKey)
}

// resolveTaskCompute resolves one task's compute to an environment. It reads the
// serverless environment version directly (no fallback) or the classic cluster's
// Spark version, consulting the job-level environments and job_clusters the task
// references by key.
func (c sdkCompute) resolveTaskCompute(ctx context.Context, settings *jobs.JobSettings, task *jobs.Task, jobID, taskKey string) (sparkVersion string, isServerless bool, version string, err error) {
	// Serverless task: it references one of the job's environments by key; the
	// version lives on that environment's spec and is used directly.
	if task.EnvironmentKey != "" {
		for _, e := range settings.Environments {
			if e.EnvironmentKey == task.EnvironmentKey {
				v := environmentVersion(e)
				if v == "" {
					return "", false, "", fmt.Errorf("task %q of job %s binds environment %q, which records no environment version", taskKey, jobID, task.EnvironmentKey)
				}
				return "", true, v, nil
			}
		}
		return "", false, "", fmt.Errorf("task %q of job %s references environment %q, which the job does not define", taskKey, jobID, task.EnvironmentKey)
	}

	// Classic task with an inline cluster spec.
	if task.NewCluster != nil && task.NewCluster.SparkVersion != "" {
		return task.NewCluster.SparkVersion, false, task.NewCluster.SparkVersion, nil
	}

	// Classic task referencing a shared job cluster by key: read that cluster's
	// Spark version from the job's job_clusters (the common classic-task shape).
	if task.JobClusterKey != "" {
		for _, jc := range settings.JobClusters {
			if jc.JobClusterKey == task.JobClusterKey {
				if jc.NewCluster.SparkVersion == "" {
					return "", false, "", fmt.Errorf("task %q of job %s uses job cluster %q, which has no spark_version", taskKey, jobID, task.JobClusterKey)
				}
				return jc.NewCluster.SparkVersion, false, jc.NewCluster.SparkVersion, nil
			}
		}
		return "", false, "", fmt.Errorf("task %q of job %s references job cluster %q, which the job does not define", taskKey, jobID, task.JobClusterKey)
	}

	// Classic task pinned to an existing cluster: resolve its Spark version.
	if task.ExistingClusterId != "" {
		sv, cerr := c.GetClusterSparkVersion(ctx, task.ExistingClusterId)
		if cerr != nil {
			return "", false, "", fmt.Errorf("resolving existing cluster %s for task %q of job %s: %w", task.ExistingClusterId, taskKey, jobID, cerr)
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
	if e.Spec.EnvironmentVersion != "" {
		return e.Spec.EnvironmentVersion
	}
	return e.Spec.Client
}
