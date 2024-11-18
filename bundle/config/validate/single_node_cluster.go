package validate

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

// Validates that any single node clusters defined in the bundle are correctly configured.
func SingleNodeCluster() bundle.ReadOnlyMutator {
	return &singleNodeCluster{}
}

type singleNodeCluster struct{}

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

func validateSingleNodeCluster(spec *compute.ClusterSpec, l []dyn.Location, p dyn.Path) *diag.Diagnostic {
	if spec == nil {
		return nil
	}

	if spec.NumWorkers > 0 || spec.Autoscale != nil {
		return nil
	}

	if spec.PolicyId != "" {
		return nil
	}

	invalidSingleNodeWarning := &diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   singleNodeWarningSummary,
		Detail:    singleNodeWarningDetail,
		Locations: l,
		Paths:     []dyn.Path{p},
	}
	profile, ok := spec.SparkConf["spark.databricks.cluster.profile"]
	if !ok {
		return invalidSingleNodeWarning
	}
	master, ok := spec.SparkConf["spark.master"]
	if !ok {
		return invalidSingleNodeWarning
	}
	resourceClass, ok := spec.CustomTags["ResourceClass"]
	if !ok {
		return invalidSingleNodeWarning
	}

	if profile == "singleNode" && strings.HasPrefix(master, "local") && resourceClass == "SingleNode" {
		return nil
	}

	return invalidSingleNodeWarning
}

func validateSingleNodePipelineCluster(spec pipelines.PipelineCluster, l []dyn.Location, p dyn.Path) *diag.Diagnostic {
	if spec.NumWorkers > 0 || spec.Autoscale != nil {
		return nil
	}

	if spec.PolicyId != "" {
		return nil
	}

	invalidSingleNodeWarning := &diag.Diagnostic{
		Severity:  diag.Warning,
		Summary:   singleNodeWarningSummary,
		Detail:    singleNodeWarningDetail,
		Locations: l,
		Paths:     []dyn.Path{p},
	}
	profile, ok := spec.SparkConf["spark.databricks.cluster.profile"]
	if !ok {
		return invalidSingleNodeWarning
	}
	master, ok := spec.SparkConf["spark.master"]
	if !ok {
		return invalidSingleNodeWarning
	}
	resourceClass, ok := spec.CustomTags["ResourceClass"]
	if !ok {
		return invalidSingleNodeWarning
	}

	if profile == "singleNode" && strings.HasPrefix(master, "local") && resourceClass == "SingleNode" {
		return nil
	}

	return invalidSingleNodeWarning
}

func (m *singleNodeCluster) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Interactive clusters
	for k, r := range rb.Config().Resources.Clusters {
		p := dyn.NewPath(dyn.Key("resources"), dyn.Key("clusters"), dyn.Key(k))
		l := rb.Config().GetLocations("resources.clusters." + k)

		d := validateSingleNodeCluster(r.ClusterSpec, l, p)
		if d != nil {
			diags = append(diags, *d)
		}
	}

	// Job clusters
	for jobK, jobV := range rb.Config().Resources.Jobs {
		for i, clusterV := range jobV.JobSettings.JobClusters {
			p := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key(jobK), dyn.Key("job_clusters"), dyn.Index(i))
			l := rb.Config().GetLocations(fmt.Sprintf("resources.jobs.%s.job_clusters[%d]", jobK, i))

			d := validateSingleNodeCluster(&clusterV.NewCluster, l, p)
			if d != nil {
				diags = append(diags, *d)
			}
		}
	}

	// Job task clusters
	for jobK, jobV := range rb.Config().Resources.Jobs {
		for i, taskV := range jobV.JobSettings.Tasks {
			if taskV.NewCluster == nil {
				continue
			}

			p := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"), dyn.Key(jobK), dyn.Key("tasks"), dyn.Index(i), dyn.Key("new_cluster"))
			l := rb.Config().GetLocations(fmt.Sprintf("resources.jobs.%s.tasks[%d].new_cluster", jobK, i))

			d := validateSingleNodeCluster(taskV.NewCluster, l, p)
			if d != nil {
				diags = append(diags, *d)
			}
		}
	}

	// Pipeline clusters
	for pipelineK, pipelineV := range rb.Config().Resources.Pipelines {
		for i, clusterV := range pipelineV.PipelineSpec.Clusters {
			p := dyn.NewPath(dyn.Key("resources"), dyn.Key("pipelines"), dyn.Key(pipelineK), dyn.Key("clusters"), dyn.Index(i))
			l := rb.Config().GetLocations(fmt.Sprintf("resources.pipelines.%s.clusters[%d]", pipelineK, i))

			d := validateSingleNodePipelineCluster(clusterV, l, p)
			if d != nil {
				diags = append(diags, *d)
			}
		}
	}

	return diags
}
