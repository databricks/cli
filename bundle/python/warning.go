package python

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

type wrapperWarning struct {
}

func WrapperWarning() bundle.Mutator {
	return &wrapperWarning{}
}

func (m *wrapperWarning) Apply(ctx context.Context, b *bundle.Bundle) error {
	if hasIncompatibleWheelTasks(ctx, b) {
		cmdio.LogString(ctx, "Python wheel tasks with local libraries require compute with DBR 13.1+. Please change your cluster configuration or set experimental 'python_wheel_wrapper' setting to 'true'")
	}
	return nil
}

func hasIncompatibleWheelTasks(ctx context.Context, b *bundle.Bundle) bool {
	tasks := libraries.FindAllWheelTasksWithLocalLibraries(b)
	for _, task := range tasks {
		if task.NewCluster != nil {
			if compare(ctx, task.NewCluster.SparkVersion) {
				return true
			}
		}

		if task.JobClusterKey != "" {
			for _, job := range b.Config.Resources.Jobs {
				for _, cluster := range job.JobClusters {
					if task.JobClusterKey == cluster.JobClusterKey && cluster.NewCluster != nil {
						if compare(ctx, cluster.NewCluster.SparkVersion) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

func compare(ctx context.Context, sparkVersion string) bool {
	version, err := extractVersion(sparkVersion)
	if err != nil {
		log.Errorf(ctx, "failed to parse spark version %s", sparkVersion)
		return false
	}
	if version < 13.1 {
		return true
	}

	return false
}

func extractVersion(sparkVersion string) (float64, error) {
	parts := strings.Split(sparkVersion, ".")
	if len(parts) < 2 {
		return 0, fmt.Errorf("failed to parse %s", sparkVersion)
	}

	v := parts[0] + "." + parts[1]
	return strconv.ParseFloat(v, 64)
}

// Name implements bundle.Mutator.
func (m *wrapperWarning) Name() string {
	return "PythonWrapperWarning"
}
