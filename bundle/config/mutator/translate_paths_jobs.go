package mutator

import (
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
)

func (m *translatePaths) applyJobTranslations(b *bundle.Bundle, v dyn.Value) (dyn.Value, error) {
	var fallback = make(map[string]string)
	var ignore []string
	var err error

	for key, job := range b.Config.Resources.Jobs {
		dir, err := job.ConfigFileDirectory()
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("unable to determine directory for job %s: %w", key, err)
		}

		// If we cannot resolve the relative path using the [dyn.Value] location itself,
		// use the job's location as fallback. This is necessary for backwards compatibility.
		fallback[key] = dir

		// Do not translate job task paths if using git source
		if job.GitSource != nil {
			ignore = append(ignore, key)
		}
	}

	// Base pattern to match all tasks in all jobs.
	base := dyn.NewPattern(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.AnyKey(),
		dyn.Key("tasks"),
		dyn.AnyIndex(),
	)

	for _, t := range []struct {
		pattern dyn.Pattern
		fn      rewriteFunc
	}{
		{
			base.Append(dyn.Key("notebook_task"), dyn.Key("notebook_path")),
			translateNotebookPath,
		},
		{
			base.Append(dyn.Key("spark_python_task"), dyn.Key("python_file")),
			translateFilePath,
		},
		{
			base.Append(dyn.Key("dbt_task"), dyn.Key("project_directory")),
			translateDirectoryPath,
		},
		{
			base.Append(dyn.Key("sql_task"), dyn.Key("file"), dyn.Key("path")),
			translateFilePath,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("whl")),
			translateNoOp,
		},
		{
			base.Append(dyn.Key("libraries"), dyn.AnyIndex(), dyn.Key("jar")),
			translateNoOp,
		},
	} {
		v, err = dyn.MapByPattern(v, t.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			key := p[2].Key()

			// Skip path translation if the job is using git source.
			if slices.Contains(ignore, key) {
				return v, nil
			}

			dir, err := v.Location().Directory()
			if err != nil {
				return dyn.InvalidValue, fmt.Errorf("unable to determine directory for job %s: %w", key, err)
			}

			return m.rewriteRelativeTo(b, p, v, t.fn, []string{
				dir,
				fallback[key],
			})
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return v, nil
}
