package localenv

import (
	"context"
	"fmt"
	"strconv"

	databricks "github.com/databricks/databricks-sdk-go"
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
		return "", true, "", nil
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
