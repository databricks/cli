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
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	sdkclusters "github.com/databricks/sdk-go/clusters/v2"
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
		taskKey := utils.Deref(task.TaskKey)
		if task.NewCluster != nil {
			if sparkVersion := utils.Deref(task.NewCluster.SparkVersion); lowerThanExpectedVersion(sparkVersion) {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("task %s uses cluster with incompatible DBR version %s", taskKey, sparkVersion),
				})
				continue
			}
		}

		if jobClusterKey := utils.Deref(task.JobClusterKey); jobClusterKey != "" {
			for _, job := range b.Config.Resources.Jobs {
				for _, cluster := range job.JobClusters {
					if cluster.NewCluster == nil {
						continue
					}
					sparkVersion := utils.Deref(cluster.NewCluster.SparkVersion)
					if jobClusterKey == utils.Deref(cluster.JobClusterKey) && sparkVersion != "" {
						if lowerThanExpectedVersion(sparkVersion) {
							diags = append(diags, diag.Diagnostic{
								Severity: diag.Error,
								Summary:  fmt.Sprintf("job cluster %s uses incompatible DBR version %s", jobClusterKey, sparkVersion),
							})
							continue
						}
					}
				}
			}
		}

		if existingClusterId := utils.Deref(task.ExistingClusterId); existingClusterId != "" {
			var version string
			var err error
			// If the cluster id is a variable and it's not resolved, it means it references a cluster defined in the same bundle.
			// So we can get the version from the cluster definition.
			// It's defined in a form of resources.clusters.<cluster_key>.id
			if strings.HasPrefix(existingClusterId, "${") {
				p, ok := dynvar.PureReferenceToPath(existingClusterId)
				if !ok || len(p) < 3 {
					log.Warnf(ctx, "unable to parse cluster key from %s", existingClusterId)
					continue
				}

				if p[0].Key() != "resources" || p[1].Key() != "clusters" {
					log.Warnf(ctx, "incorrect variable reference for cluster id %s", existingClusterId)
					continue
				}

				clusterKey := p[2].Key()
				cluster, ok := b.Config.Resources.Clusters[clusterKey]
				if !ok {
					log.Warnf(ctx, "unable to find cluster with key %s", clusterKey)
					continue
				}
				version = utils.Deref(cluster.SparkVersion)
			} else {
				version, err = getSparkVersionForCluster(ctx, b.WorkspaceClient(ctx), existingClusterId)
				// If there's error getting spark version for cluster, do not mark it as incompatible
				if err != nil {
					log.Warnf(ctx, "unable to get spark version for cluster %s, err: %s", existingClusterId, err.Error())
					continue
				}
			}

			if lowerThanExpectedVersion(version) {
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("task %s uses cluster with incompatible DBR version %s", taskKey, version),
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

func getSparkVersionForCluster(ctx context.Context, w *sdkclients.Clients, clusterId string) (string, error) {
	details, err := w.Clusters.GetCluster(ctx, sdkclusters.GetClusterRequest{ClusterId: new(clusterId)})
	if err != nil {
		return "", err
	}

	return utils.Deref(details.SparkVersion), nil
}
