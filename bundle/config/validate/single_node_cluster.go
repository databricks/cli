package validate

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// Validates that any single node clusters defined in the bundle are correctly configured.
func SingleNodeCluster() bundle.ReadOnlyMutator {
	return &singleNodeCluster{}
}

type singleNodeCluster struct{ bundle.RO }

func (m *singleNodeCluster) Name() string {
	return "validate:SingleNodeCluster"
}

const singleNodeWarningDetail = `num_workers should be 0 only for single-node clusters. To create a
valid single node cluster please ensure that the following properties
are correctly set in the cluster specification:

  spark_conf:
    spark.databricks.cluster.profile: singleNode
    spark.master: local[*]

  custom_tags:
    ResourceClass: SingleNode
  `

const singleNodeWarningSummary = `Single node cluster is not correctly configured`

// clusterSpec holds the fields of compute.ClusterSpec and pipelines.PipelineCluster
// that the validation reads.
type clusterSpec struct {
	numWorkers    int
	numWorkersSet bool
	isSingleNode  bool
	sparkConf     map[string]string
	customTags    map[string]string
	policyId      string
}

func computeClusterSpec(c *compute.ClusterSpec) clusterSpec {
	return clusterSpec{
		numWorkers:    c.NumWorkers,
		numWorkersSet: c.NumWorkers != 0 || slices.Contains(c.ForceSendFields, "NumWorkers"),
		isSingleNode:  c.IsSingleNode,
		sparkConf:     c.SparkConf,
		customTags:    c.CustomTags,
		policyId:      c.PolicyId,
	}
}

func pipelineClusterSpec(c *pipelines.PipelineCluster) clusterSpec {
	return clusterSpec{
		numWorkers:    c.NumWorkers,
		numWorkersSet: c.NumWorkers != 0 || slices.Contains(c.ForceSendFields, "NumWorkers"),
		sparkConf:     c.SparkConf,
		customTags:    c.CustomTags,
		policyId:      c.PolicyId,
	}
}

// showSingleNodeClusterWarning reports whether the cluster c at p sets num_workers to
// 0 without configuring a single node cluster.
func showSingleNodeClusterWarning(ctx context.Context, b *bundle.Bundle, p *structpath.PathNode, c clusterSpec) bool {
	// Check if the user has explicitly set the num_workers to 0. Skip the warning
	// if that's not the case.
	if !c.numWorkersSet || c.numWorkers > 0 {
		return false
	}

	// If is_single_node is set to true, the cluster is correctly configured automatically.
	// No need to show the warning.
	if c.isSingleNode {
		return false
	}

	// Values that are pure references are not resolved yet.
	for _, k := range []string{"num_workers", "spark_conf", "custom_tags"} {
		if b.Config.IsReferenceAt(structpath.NewStringKey(p, k)) {
			return false
		}
	}

	// If the policy id is set, we don't want to show the warning. This is because
	// the user might have configured `spark_conf` and `custom_tags` correctly
	// in their cluster policy.
	if c.policyId != "" {
		return false
	}

	profile, ok := c.sparkConf["spark.databricks.cluster.profile"]
	if !ok {
		log.Debugf(ctx, "spark_conf spark.databricks.cluster.profile not found in single-node cluster spec")
		return true
	}
	if profile != "singleNode" {
		log.Debugf(ctx, "spark_conf spark.databricks.cluster.profile is not singleNode in single-node cluster spec: %s", profile)
		return true
	}

	master, ok := c.sparkConf["spark.master"]
	if !ok {
		log.Debugf(ctx, "spark_conf spark.master not found in single-node cluster spec")
		return true
	}
	if !strings.HasPrefix(master, "local") {
		log.Debugf(ctx, "spark_conf spark.master does not start with local in single-node cluster spec: %s", master)
		return true
	}

	resourceClass, ok := c.customTags["ResourceClass"]
	if !ok {
		log.Debugf(ctx, "custom_tag ResourceClass not found in single-node cluster spec")
		return true
	}
	if resourceClass != "SingleNode" {
		log.Debugf(ctx, "custom_tag ResourceClass is not SingleNode in single-node cluster spec: %s", resourceClass)
		return true
	}

	return false
}

func (m *singleNodeCluster) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var specs clusterSpecs
	r := &b.Config.Resources

	// Interactive clusters
	for _, name := range slices.Sorted(maps.Keys(r.Clusters)) {
		if c := r.Clusters[name]; c != nil {
			specs.add(structpath.NewPath(nil, "resources", "clusters", name), computeClusterSpec(&c.ClusterSpec))
		}
	}
	// Job clusters
	for _, name := range slices.Sorted(maps.Keys(r.Jobs)) {
		if job := r.Jobs[name]; job != nil {
			for i := range job.JobClusters {
				if nc := job.JobClusters[i].NewCluster; nc != nil {
					specs.add(structpath.NewPath(nil, "resources", "jobs", name, "job_clusters", i, "new_cluster"), computeClusterSpec(nc))
				}
			}
		}
	}
	// Job task clusters, then the clusters of tasks in for_each_task
	for _, nested := range []bool{false, true} {
		r.IterateJobTasks(nested, specs.task)
	}
	// Pipeline clusters
	for _, name := range slices.Sorted(maps.Keys(r.Pipelines)) {
		if pl := r.Pipelines[name]; pl != nil {
			for i := range pl.Clusters {
				specs.add(structpath.NewPath(nil, "resources", "pipelines", name, "clusters", i), pipelineClusterSpec(&pl.Clusters[i]))
			}
		}
	}
	diags := diag.Diagnostics{}
	for _, c := range specs {
		if showSingleNodeClusterWarning(ctx, b, c.path, c.spec) {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   singleNodeWarningSummary,
				Detail:    singleNodeWarningDetail,
				Locations: b.Config.GetLocationsOfExact(c.path),
				Paths:     []*structpath.PathNode{c.path},
			})
		}
	}
	return diags
}

// clusterSpecAt is a cluster spec to check and its path.
type clusterSpecAt struct {
	path *structpath.PathNode
	spec clusterSpec
}

// clusterSpecs collects the cluster specs to check, in the order of the diagnostics.
type clusterSpecs []clusterSpecAt

func (c *clusterSpecs) add(p *structpath.PathNode, spec clusterSpec) {
	*c = append(*c, clusterSpecAt{p, spec})
}

func (c *clusterSpecs) task(p *structpath.PathNode, task *jobs.Task) bool {
	if task.NewCluster != nil {
		c.add(structpath.NewPath(p, "new_cluster"), computeClusterSpec(task.NewCluster))
	}
	return true
}
