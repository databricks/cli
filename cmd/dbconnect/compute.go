package dbconnect

import (
	"context"
	"fmt"
	"strconv"

	databricks "github.com/databricks/databricks-sdk-go"
)

// sdkCompute adapts the Databricks SDK to the dbconnect.ComputeClient interface.
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
// If neither indicator is present the job's compute cannot be determined.
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

	// Serverless jobs have Environments populated; classic compute uses JobClusters.
	if len(job.Settings.Environments) > 0 {
		return "", true, "", nil
	}

	if len(job.Settings.JobClusters) > 0 {
		sv := job.Settings.JobClusters[0].NewCluster.SparkVersion
		if sv == "" {
			return "", false, "", fmt.Errorf("could not determine compute for job %d: first job cluster has no spark_version", id)
		}
		return sv, false, sv, nil
	}

	return "", false, "", fmt.Errorf("could not determine compute for job %d: no environments or job clusters found", id)
}
