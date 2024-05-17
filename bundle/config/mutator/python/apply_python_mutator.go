package python

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
)

type phase string

const (
	ApplyPythonMutatorPhasePreInit phase = "preinit"
	ApplyPythonMutatorPhaseInit    phase = "init"
)

type applyPythonMutator struct {
	phase phase
}

func ApplyPythonMutator(phase phase) bundle.Mutator {
	return &applyPythonMutator{
		phase: phase,
	}
}

func (m *applyPythonMutator) Name() string {
	return fmt.Sprintf("ApplyPythonMutator(%s)", m.phase)
}

func getExperimental(b *bundle.Bundle) config.Experimental {
	if b.Config.Experimental == nil {
		return config.Experimental{}
	}

	return *b.Config.Experimental
}

func (m *applyPythonMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	experimental := getExperimental(b)

	if !experimental.PyDABs.Enable {
		log.Debugf(ctx, "'experimental.pydabs.enable' isn't enabled, skipping")
		return nil
	}

	if experimental.PyDABs.Enable && experimental.VEnv.Path == "" {
		return diag.Errorf("'experimental.pydabs.enable' can only be used when 'experimental.venv.path' is set")
	}

	err := b.Config.Mutate(func(leftRoot dyn.Value) (dyn.Value, error) {
		pythonPath := path.Join(experimental.VEnv.Path, "bin", "python3")

		if _, err := os.Stat(pythonPath); err != nil {
			if os.IsNotExist(err) {
				return dyn.InvalidValue, fmt.Errorf("can't find '%s', check if venv is created", pythonPath)
			} else {
				return dyn.InvalidValue, fmt.Errorf("can't find '%s': %w", pythonPath, err)
			}
		}

		rightRoot, err := m.runPythonMutator(ctx, b.RootPath, pythonPath, leftRoot)

		if err != nil {
			return dyn.InvalidValue, err
		}

		return merge.Override(leftRoot, rightRoot, createOverrideVisitor(ctx, m.phase))
	})

	return diag.FromErr(err)
}

func (m *applyPythonMutator) runPythonMutator(ctx context.Context, rootPath string, pythonPath string, root dyn.Value) (dyn.Value, error) {
	args := []string{
		pythonPath,
		"-m",
		"databricks.bundles.build",
		"--phase",
		string(m.phase),
	}

	// we need to marshal dyn.Value instead of bundle.Config to JSON to support
	// non-string fields assigned with bundle variables
	rootConfigJson, err := json.Marshal(root.AsAny())

	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to marshal root config: %w", err)
	}

	logWriter := newLogWriter(ctx, "stderr: ")

	stdout, err := process.Background(
		ctx,
		args,
		process.WithDir(rootPath),
		process.WithStderrWriter(logWriter),
		process.WithStdinReader(bytes.NewBuffer(rootConfigJson)),
	)

	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("python mutator process failed: %w", err)
	}

	// we need absolute path, or because later parts of pipeline assume all paths are absolute
	// and this file will be used as location
	virtualPath, err := filepath.Abs(filepath.Join(rootPath, "__generated__.yml"))

	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to get absolute path: %w", err)
	}

	generated, err := yamlloader.LoadYAML(virtualPath, bytes.NewReader([]byte(stdout)))

	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to parse Python mutator output: %w", err)
	}

	normalized, diagnostic := convert.Normalize(config.Root{}, generated)

	if diagnostic.Error() != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to normalize Python mutator output: %w", diagnostic.Error())
	}

	return normalized, nil
}
func createOverrideVisitor(ctx context.Context, phase phase) merge.OverrideVisitor {
	jobsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"))

	return merge.OverrideVisitor{
		VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
			if !valuePath.HasPrefix(jobsPath) {
				return fmt.Errorf("unexpected change at '%s' (delete)", valuePath.String())
			}

			deleteResource := len(valuePath) == len(jobsPath)+1

			if phase == ApplyPythonMutatorPhasePreInit {
				return fmt.Errorf("unexpected change at '%s' (delete)", valuePath.String())
			} else if deleteResource && phase == ApplyPythonMutatorPhaseInit {
				return fmt.Errorf("unexpected change at '%s' (delete)", valuePath.String())
			}

			log.Debugf(ctx, "Delete value at '%s'", valuePath.String())

			return nil
		},
		VisitInsert: func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			if !valuePath.HasPrefix(jobsPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at '%s' (insert)", valuePath.String())
			}

			insertResource := len(valuePath) == len(jobsPath)+1

			if phase == ApplyPythonMutatorPhasePreInit && !insertResource {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at '%s' (insert)", valuePath.String())
			}

			log.Debugf(ctx, "Insert value at '%s'", valuePath.String())

			return right, nil
		},
		VisitUpdate: func(valuePath dyn.Path, left dyn.Value, right dyn.Value) (dyn.Value, error) {
			if !valuePath.HasPrefix(jobsPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at '%s' (update)", valuePath.String())
			}

			if phase == ApplyPythonMutatorPhasePreInit {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at '%s' (update)", valuePath.String())
			}

			log.Debugf(ctx, "Update value at '%s'", valuePath.String())

			return right, nil
		},
	}
}
