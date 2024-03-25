package python

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/mod/semver"
)

type wrapperWarning struct {
}

func WrapperWarning() bundle.Mutator {
	return &wrapperWarning{}
}

func (m *wrapperWarning) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if isPythonWheelWrapperOn(b) {
		return nil
	}

	if hasIncompatibleWheelTasks(ctx, b) {
		return diag.Errorf("python wheel tasks with local libraries require compute with DBR 13.1+. Please change your cluster configuration or set experimental 'python_wheel_wrapper' setting to 'true'")
	}
	return nil
}

func isPythonWheelWrapperOn(b *bundle.Bundle) bool {
	return b.Config.Experimental != nil && b.Config.Experimental.PythonWheelWrapper
}

func hasIncompatibleWheelTasks(ctx context.Context, b *bundle.Bundle) bool {
	tasks := libraries.FindAllWheelTasksWithLocalLibraries(b)
	for _, task := range tasks {
		if task.NewCluster != nil {
			if lowerThanExpectedVersion(ctx, task.NewCluster.SparkVersion) {
				return true
			}
		}

		if task.JobClusterKey != "" {
			for _, job := range b.Config.Resources.Jobs {
				for _, cluster := range job.JobClusters {
					if task.JobClusterKey == cluster.JobClusterKey && cluster.NewCluster != nil {
						if lowerThanExpectedVersion(ctx, cluster.NewCluster.SparkVersion) {
							return true
						}
					}
				}
			}
		}

		if task.ExistingClusterId != "" {
			version, err := getSparkVersionForCluster(ctx, b.WorkspaceClient(), task.ExistingClusterId)

			// If there's error getting spark version for cluster, do not mark it as incompatible
			if err != nil {
				log.Warnf(ctx, "unable to get spark version for cluster %s, err: %s", task.ExistingClusterId, err.Error())
				return false
			}

			if lowerThanExpectedVersion(ctx, version) {
				return true
			}
		}
	}

	return false
}

func lowerThanExpectedVersion(ctx context.Context, sparkVersion string) bool {
	parts := strings.Split(sparkVersion, ".")
	if len(parts) < 2 {
		return false
	}

	if parts[1][0] == 'x' { // treat versions like 13.x as the very latest minor (13.99)
		parts[1] = "99"
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
