package mutator

import (
	"context"
	"strings"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/databricks-sdk-go/service/libraries"
)

type defaultDbtTaskLibrary struct{}

// The version of "dbt-databricks" to inject as default dependency for DBT tasks.
const DbtDatabricksVersionSpec = "dbt-databricks>=1.0.0,<2.0.0"

// DefaultDbtTaskLibrary adds dependency on "dbt-databricks" to DBT tasks if not yet present.
func DefaultDbtTaskLibrary() bundle.Mutator {
	return &defaultDbtTaskLibrary{}
}

func (m *defaultDbtTaskLibrary) Name() string {
	return "DefaultDbtTaskLibrary"
}

func (m *defaultDbtTaskLibrary) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	jobs := b.Config.Resources.Jobs
	for k, v := range jobs {
		for i := 0; i < len(v.Tasks); i++ {
			task := &v.Tasks[i]
			if task.DbtTask == nil {
				continue
			}

			// The DBT task is set.
			// Check if the task's libraries includes "dbt-databricks".
			hasDependency := false
			for _, library := range task.Libraries {
				if library.Pypi == nil {
					continue
				}

				if strings.HasPrefix(library.Pypi.Package, "dbt-databricks") {
					hasDependency = true
					break
				}
			}

			// Add library if it isn't yet included.
			if !hasDependency {
				task.Libraries = append(task.Libraries, libraries.Library{
					Pypi: &libraries.PythonPyPiLibrary{
						Package: "dbt-databricks>=1.0.0,<2.0.0",
					},
				})
			}
		}

		jobs[k] = v
	}

	return nil, nil
}
