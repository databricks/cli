package trampoline

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/mod/semver"
)

type wrapperWarning struct{}

func WrapperWarning() bundle.Mutator {
	return &wrapperWarning{}
}

func (m *wrapperWarning) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if isPythonWheelWrapperOn(b) {
		if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
			return diag.Warningf("Python wheel notebook wrapper is not available when using source-linked deployment mode. You can disable this mode by setting 'presets.source_linked_deployment: false'")
		}
		return nil
	}

	diags := hasIncompatibleWheelTasks(ctx, b)
	if len(diags) > 0 {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Python wheel tasks require compute with DBR 13.3+ to include local libraries. Please change your cluster configuration or use the experimental 'python_wheel_wrapper' setting. See https://docs.databricks.com/dev-tools/bundles/python-wheel.html for more information.",
		})
	}
	return diags
}

func isPythonWheelWrapperOn(b *bundle.Bundle) bool {
	return b.Config.Experimental != nil && b.Config.Experimental.PythonWheelWrapper
}

func hasIncompatibleWheelTasks(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	tasks := libraries.FindTasksWithLocalLibraries(b)
	for _, task := range tasks {
		if task.NewCluster != nil {
			if lowerThanExpectedVersion(task.NewCluster.SparkVersion) {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("task %s uses cluster with incompatible DBR version %s", task.TaskKey, task.NewCluster.SparkVersion),
				})
				continue
			}
		}

		if task.JobClusterKey != "" {
			for _, job := range b.Config.Resources.Jobs {
				for _, cluster := range job.JobClusters {
					if task.JobClusterKey == cluster.JobClusterKey && cluster.NewCluster.SparkVersion != "" {
						if lowerThanExpectedVersion(cluster.NewCluster.SparkVersion) {
							diags = append(diags, diag.Diagnostic{
								Severity: diag.Error,
								Summary:  fmt.Sprintf("job cluster %s uses incompatible DBR version %s", cluster.JobClusterKey, cluster.NewCluster.SparkVersion),
							})
							continue
						}
					}
				}
			}
		}

		if task.ExistingClusterId != "" {
			var version string
			var err error
			// If the cluster id is a variable and it's not resolved, it means it references a cluster defined in the same bundle.
			// So we can get the version from the cluster definition.
			// It's defined in a form of resources.clusters.<cluster_key>.id
			if strings.HasPrefix(task.ExistingClusterId, "${") {
				p, ok := dynvar.PureReferenceToPath(task.ExistingClusterId)
				if !ok || len(p) < 3 {
					log.Warnf(ctx, "unable to parse cluster key from %s", task.ExistingClusterId)
					continue
				}

				if p[0].Key() != "resources" || p[1].Key() != "clusters" {
					log.Warnf(ctx, "incorrect variable reference for cluster id %s", task.ExistingClusterId)
					continue
				}

				clusterKey := p[2].Key()
				cluster, ok := b.Config.Resources.Clusters[clusterKey]
				if !ok {
					log.Warnf(ctx, "unable to find cluster with key %s", clusterKey)
					continue
				}
				version = cluster.SparkVersion
			} else {
				version, err = getSparkVersionForCluster(ctx, b.WorkspaceClient(), task.ExistingClusterId)
				// If there's error getting spark version for cluster, do not mark it as incompatible
				if err != nil {
					log.Warnf(ctx, "unable to get spark version for cluster %s, err: %s", task.ExistingClusterId, err.Error())
					continue
				}
			}

			if lowerThanExpectedVersion(version) {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("task %s uses cluster with incompatible DBR version %s", task.TaskKey, version),
				})
				continue
			}
		}
	}

	return diags
}

func lowerThanExpectedVersion(sparkVersion string) bool {
	parts := strings.Split(sparkVersion, ".")
	if len(parts) < 2 {
		return false
	}

	if len(parts[1]) > 0 && parts[1][0] == 'x' { // treat versions like 13.x as the very latest minor (13.99)
		parts[1] = "99"
	}

	// if any of the version parts are not numbers, we can't compare
	// so consider it as compatible version
	if _, err := strconv.Atoi(parts[0]); err != nil {
		return false
	}

	if _, err := strconv.Atoi(parts[1]); err != nil {
		return false
	}

	v := "v" + parts[0] + "." + parts[1]
	return semver.Compare(v, "v13.1") < 0
}

// Name implements bundle.Mutator.
func (m *wrapperWarning) Name() string {
	return "PythonWrapperWarning"
}

func getSparkVersionForCluster(ctx context.Context, w *databricks.WorkspaceClient, clusterId string) (string, error) {
	details, err := w.Clusters.GetByClusterId(ctx, clusterId)
	if err != nil {
		return "", err
	}

	return details.SparkVersion, nil
}
