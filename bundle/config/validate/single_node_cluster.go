package validate

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

// stringMap reads a view of a map of strings. It returns false if v is not a map.
func stringMap(v structvar.View) (map[string]string, bool) {
	switch v.Kind() {
	case structvar.KindInvalid, structvar.KindNil:
		return nil, true
	case structvar.KindMap:
	default:
		return nil, false
	}
	out := map[string]string{}
	for k, c := range v.MapItems() {
		s, ok := c.AsString()
		if !ok {
			return nil, false
		}
		out[k] = s
	}
	return out, true
}

func showSingleNodeClusterWarning(ctx context.Context, v structvar.View) bool {
	// Check if the user has explicitly set the num_workers to 0. Skip the warning
	// if that's not the case.
	numWorkers, ok := v.Get("num_workers").AsInt()
	if !ok || numWorkers > 0 {
		return false
	}

	// If is_single_node is set to true, the cluster is correctly configured automatically.
	// No need to show the warning.
	isSingleNode, ok := v.Get("is_single_node").AsBool()
	if ok && isSingleNode {
		return false
	}

	// Read the common fields from compute.ClusterSpec and pipelines.PipelineCluster
	// that we are interested in.
	sparkConf, ok := stringMap(v.Get("spark_conf"))
	if !ok {
		return false
	}
	customTags, ok := stringMap(v.Get("custom_tags"))
	if !ok {
		return false
	}
	policyId, _ := v.Get("policy_id").AsString()

	// If the policy id is set, we don't want to show the warning. This is because
	// the user might have configured `spark_conf` and `custom_tags` correctly
	// in their cluster policy.
	if policyId != "" {
		return false
	}

	profile, ok := sparkConf["spark.databricks.cluster.profile"]
	if !ok {
		log.Debugf(ctx, "spark_conf spark.databricks.cluster.profile not found in single-node cluster spec")
		return true
	}
	if profile != "singleNode" {
		log.Debugf(ctx, "spark_conf spark.databricks.cluster.profile is not singleNode in single-node cluster spec: %s", profile)
		return true
	}

	master, ok := sparkConf["spark.master"]
	if !ok {
		log.Debugf(ctx, "spark_conf spark.master not found in single-node cluster spec")
		return true
	}
	if !strings.HasPrefix(master, "local") {
		log.Debugf(ctx, "spark_conf spark.master does not start with local in single-node cluster spec: %s", master)
		return true
	}

	resourceClass, ok := customTags["ResourceClass"]
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

	patterns := []*structpath.PatternNode{
		// Interactive clusters
		structpath.MustParsePattern("resources.clusters.*"),
		// Job clusters
		structpath.MustParsePattern("resources.jobs.*.job_clusters[*].new_cluster"),
		// Job task clusters
		structpath.MustParsePattern("resources.jobs.*.tasks[*].new_cluster"),
		// Job for each task clusters
		structpath.MustParsePattern("resources.jobs.*.tasks[*].for_each_task.task.new_cluster"),
		// Pipeline clusters
		structpath.MustParsePattern("resources.pipelines.*.clusters[*]"),
	}

	root := b.Config.View()
	for _, p := range patterns {
		err := structvar.ForEach(root, p, func(np *structpath.PathNode, v structvar.View) error {
			warning := diag.Diagnostic{
				Severity:  diag.Warning,
				Summary:   singleNodeWarningSummary,
				Detail:    singleNodeWarningDetail,
				Locations: v.Locations(),
				Paths:     []*structpath.PathNode{np},
			}

			if showSingleNodeClusterWarning(ctx, v) {
				diags = append(diags, warning)
			}
			return nil
		})
		if err != nil {
			log.Debugf(ctx, "Error while applying single node cluster validation: %s", err)
		}
	}
	return diags
}
