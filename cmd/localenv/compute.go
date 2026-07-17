package localenv

import (
	"context"
	"fmt"
	"strconv"

	databricks "github.com/databricks/databricks-sdk-go"
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

// GetJobSparkVersion inspects the job's configuration to determine compute type.
//
// A job is considered serverless when it has non-empty Environments (JobEnvironment
// entries), which signals the Databricks serverless runtime. A job with classic compute
// uses JobClusters; we read SparkVersion from the first job cluster's NewCluster spec.
//
// Task-level compute (tasks[].new_cluster / tasks[].existing_cluster_id with no
// job-level job_clusters) is not resolved here: it may vary per task and an
// existing_cluster_id would need a second lookup, which is out of scope for the
// initial job support. Such a job returns an actionable error rather than a wrong
// guess; use --cluster or --serverless explicitly instead.
func (c sdkCompute) GetJobSparkVersion(ctx context.Context, jobID string) (sparkVersion string, isServerless bool, version string, err error) {
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

	// A job that declares both serverless environments and classic job clusters is
	// ambiguous: its tasks can run on different compute, so there is no single
	// correct local environment to provision. Refuse rather than guess serverless.
	if len(job.Settings.Environments) > 0 && len(job.Settings.JobClusters) > 0 {
		return "", false, "", fmt.Errorf("job %d has both serverless environments and job clusters; pass --cluster or --serverless explicitly to disambiguate", id)
	}

	// Serverless jobs have Environments populated; classic compute uses JobClusters.
	if len(job.Settings.Environments) > 0 {
		// The serverless environment version (e.g. "4") is recorded on the job's
		// environment spec, unlike the bundle path where it is unavailable. Return
		// it so ResolveTarget pins the matching serverless-vN instead of defaulting
		// to v4. An empty version (older jobs) falls back to v4 in ResolveTarget.
		version := environmentVersion(job.Settings.Environments[0])
		// Tasks can reference any environment_key, so if the job's environments do
		// not all share one version there is no single correct local environment
		// (mirrors the job-cluster check below). Refuse rather than guess from the
		// first. A pinned-vs-unpinned mix is also ambiguous, so compare raw values.
		for _, e := range job.Settings.Environments[1:] {
			if environmentVersion(e) != version {
				return "", false, "", fmt.Errorf("job %d has serverless environments with differing versions; pass --serverless explicitly to disambiguate", id)
			}
		}
		return "", true, version, nil
	}

	if len(job.Settings.JobClusters) > 0 {
		sv := job.Settings.JobClusters[0].NewCluster.SparkVersion
		if sv == "" {
			return "", false, "", fmt.Errorf("could not determine compute for job %d: first job cluster has no spark_version", id)
		}
		// Tasks can reference any job_cluster_key, so if the job's clusters do not
		// all share one Spark version there is no single correct local environment.
		// Refuse rather than silently provisioning for the first cluster.
		for _, jc := range job.Settings.JobClusters[1:] {
			if jc.NewCluster.SparkVersion != sv {
				return "", false, "", fmt.Errorf("job %d has job clusters with differing spark_version; pass --cluster or --serverless explicitly to disambiguate", id)
			}
		}
		return sv, false, sv, nil
	}

	return "", false, "", fmt.Errorf("could not determine compute for job %d from its environments or job clusters (task-level compute is not supported); pass --cluster or --serverless explicitly", id)
}

// environmentVersion returns the serverless environment version recorded on a
// job environment, or "" when the spec or version is absent.
//
// The version can arrive in either of two fields. environment_version is the
// current one; client is its deprecated predecessor ("Use environment_version
// instead") and is still what some jobs pin. Reading both means the v4 fallback
// and the divergence guard observe whichever field actually carries the pin,
// rather than treating a client-pinned job as unversioned. base_environment is
// deliberately ignored: it is a path/ID, not a version.
func environmentVersion(e jobs.JobEnvironment) string {
	if e.Spec == nil {
		return ""
	}
	if e.Spec.EnvironmentVersion != "" {
		return e.Spec.EnvironmentVersion
	}
	return e.Spec.Client
}
