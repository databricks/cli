package validate

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
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

func showSingleNodeClusterWarning(ctx context.Context, v dyn.Value) bool {
	// Check if the user has explicitly set the num_workers to 0. Skip the warning
	// if that's not the case.
	numWorkers, ok := v.Get("num_workers").AsInt()
	if !ok || numWorkers > 0 {
		return false
	}

	// Convenient type that contains the common fields from compute.ClusterSpec and
	// pipelines.PipelineCluster that we are interested in.
	type ClusterConf struct {
		SparkConf  map[string]string `json:"spark_conf"`
		CustomTags map[string]string `json:"custom_tags"`
		PolicyId   string            `json:"policy_id"`
	}

	conf := &ClusterConf{}
	err := convert.ToTyped(conf, v)
	if err != nil {
		return false
	}

	// If the policy id is set, we don't want to show the warning. This is because
	// the user might have configured `spark_conf` and `custom_tags` correctly
	// in their cluster policy.
	if conf.PolicyId != "" {
		return false
	}

	profile, ok := conf.SparkConf["spark.databricks.cluster.profile"]
	if !ok {
		log.Debugf(ctx, "spark_conf spark.databricks.cluster.profile not found in single-node cluster spec")
		return true
	}
	if profile != "singleNode" {
		log.Debugf(ctx, "spark_conf spark.databricks.cluster.profile is not singleNode in single-node cluster spec: %s", profile)
		return true
	}

	master, ok := conf.SparkConf["spark.master"]
	if !ok {
		log.Debugf(ctx, "spark_conf spark.master not found in single-node cluster spec")
		return true
	}
	if !strings.HasPrefix(master, "local") {
		log.Debugf(ctx, "spark_conf spark.master does not start with local in single-node cluster spec: %s", master)
		return true
	}

	resourceClass, ok := conf.CustomTags["ResourceClass"]
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
	diags := diag.Diagnostics{}

	patterns := []dyn.Pattern{
		// Interactive clusters
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("clusters"), dyn.AnyKey()),
		// Job clusters
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("job_clusters"), dyn.AnyIndex(), dyn.Key("new_cluster")),
		// Job task clusters
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("new_cluster")),
		// Job for each task clusters
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex(), dyn.Key("for_each_task"), dyn.Key("task"), dyn.Key("new_cluster")),
		// Pipeline clusters
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("pipelines"), dyn.AnyKey(), dyn.Key("clusters"), dyn.AnyIndex()),
	}

	for _, p := range patterns {
		_, err := dyn.MapByPattern(b.Config.Value(), p, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			warning := diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   singleNodeWarningSummary,
				Detail:    singleNodeWarningDetail,
				Locations: v.Locations(),
				Paths:     []dyn.Path{p},
			}

			if showSingleNodeClusterWarning(ctx, v) {
				diags = append(diags, warning)
			}
			return v, nil
		})
		if err != nil {
			log.Debugf(ctx, "Error while applying single node cluster validation: %s", err)
		}
	}
	return diags
}
