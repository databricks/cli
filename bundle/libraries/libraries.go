package libraries

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"golang.org/x/exp/slices"
)

type attach struct {
}

func AttachToResources() bundle.Mutator {
	return &attach{}
}

func (a *attach) Name() string {
	return "libraries.AttachToResources"
}

func (a *attach) Apply(ctx context.Context, b *bundle.Bundle) error {
	r := b.Config.Resources
	for k := range b.Config.Resources.Jobs {
		tasks := r.Jobs[k].JobSettings.Tasks
		for i := range tasks {
			task := &tasks[i]
			if task.PythonWheelTask != nil {
				packageName := task.PythonWheelTask.PackageName
				artifact, ok := b.Config.Artifacts[packageName]
				if !ok {
					// TODO: we can be even more smart about it and automatically detect potential artifact
					// (like Python wheel) and try to build it first
					return fmt.Errorf("artifact not found: %s. Please define the artifact in bundle artifacts section", packageName)
				}

				lib := artifact.Library()
				alreadyAdded := slices.ContainsFunc(task.Libraries, func(e compute.Library) bool {
					return e.Whl != "" && e.Whl == lib.Whl
				})
				if !alreadyAdded {
					log.Debugf(ctx, "Attaching %s (%s) to %s", packageName, artifact.RemotePath, task.TaskKey)
					task.Libraries = append(task.Libraries, lib)
				}
			}
		}
	}
	return nil
}
