package python

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/python"
	"github.com/databricks/databricks-sdk-go/logger"

	"github.com/databricks/cli/bundle/env"

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
	// PythonMutatorPhaseLoad is the phase in which bundle configuration is loaded.
	//
	// At this stage, PyDABs adds statically defined resources to the bundle configuration.
	// Which resources are added should be deterministic and not depend on the bundle configuration.
	//
	// We also open for possibility of appending other sections of bundle configuration,
	// for example, adding new variables. However, this is not supported yet, and CLI rejects
	// such changes.
	PythonMutatorPhaseLoad phase = "load"

	// PythonMutatorPhaseInit is the phase after bundle configuration was loaded, and
	// the list of statically declared resources is known.
	//
	// At this stage, PyDABs adds resources defined using generators, or mutates existing resources,
	// including the ones defined using YAML.
	//
	// During this process, within generator and mutators, PyDABs can access:
	// - selected deployment target
	// - bundle variables values
	// - variables provided through CLI arguments or environment variables
	//
	// The following is not available:
	// - variables referencing other variables are in unresolved format
	//
	// PyDABs can output YAML containing references to variables, and CLI should resolve them.
	//
	// Existing resources can't be removed, and CLI rejects such changes.
	PythonMutatorPhaseInit phase = "init"
)

type pythonMutator struct {
	phase phase
}

func PythonMutator(phase phase) bundle.Mutator {
	return &pythonMutator{
		phase: phase,
	}
}

func (m *pythonMutator) Name() string {
	return fmt.Sprintf("PythonMutator(%s)", m.phase)
}

func getExperimental(b *bundle.Bundle) config.Experimental {
	if b.Config.Experimental == nil {
		return config.Experimental{}
	}

	return *b.Config.Experimental
}

func (m *pythonMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	experimental := getExperimental(b)

	if !experimental.PyDABs.Enabled {
		return nil
	}

	// mutateDiags is used because Mutate returns 'error' instead of 'diag.Diagnostics'
	var mutateDiags diag.Diagnostics
	var mutateDiagsHasError = errors.New("unexpected error")

	err := b.Config.Mutate(func(leftRoot dyn.Value) (dyn.Value, error) {
		pythonPath, err := detectExecutable(ctx, experimental.PyDABs.VEnvPath)

		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to get Python interpreter path: %w", err)
		}

		cacheDir, err := createCacheDir(ctx)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to create cache dir: %w", err)
		}

		rightRoot, diags := m.runPythonMutator(ctx, cacheDir, b.RootPath, pythonPath, leftRoot)
		mutateDiags = diags
		if diags.HasError() {
			return dyn.InvalidValue, mutateDiagsHasError
		}

		visitor, err := createOverrideVisitor(ctx, m.phase)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return merge.Override(leftRoot, rightRoot, visitor)
	})

	if err == mutateDiagsHasError {
		if !mutateDiags.HasError() {
			panic("mutateDiags has no error, but error is expected")
		}

		return mutateDiags
	}

	return mutateDiags.Extend(diag.FromErr(err))
}

func createCacheDir(ctx context.Context) (string, error) {
	// b.CacheDir doesn't work because target isn't yet selected

	// support the same env variable as in b.CacheDir
	if tempDir, exists := env.TempDir(ctx); exists {
		// use 'default' as target name
		cacheDir := filepath.Join(tempDir, "default", "pydabs")

		err := os.MkdirAll(cacheDir, 0700)
		if err != nil {
			return "", err
		}

		return cacheDir, nil
	}

	return os.MkdirTemp("", "-pydabs")
}

func (m *pythonMutator) runPythonMutator(ctx context.Context, cacheDir string, rootPath string, pythonPath string, root dyn.Value) (dyn.Value, diag.Diagnostics) {
	inputPath := filepath.Join(cacheDir, "input.json")
	outputPath := filepath.Join(cacheDir, "output.json")
	diagnosticsPath := filepath.Join(cacheDir, "diagnostics.json")

	args := []string{
		pythonPath,
		"-m",
		"databricks.bundles.build",
		"--phase",
		string(m.phase),
		"--input",
		inputPath,
		"--output",
		outputPath,
		"--diagnostics",
		diagnosticsPath,
	}

	if err := writeInputFile(inputPath, root); err != nil {
		return dyn.InvalidValue, diag.Errorf("failed to write input file: %s", err)
	}

	stderrWriter := newLogWriter(ctx, "stderr: ")
	stdoutWriter := newLogWriter(ctx, "stdout: ")

	_, processErr := process.Background(
		ctx,
		args,
		process.WithDir(rootPath),
		process.WithStderrWriter(stderrWriter),
		process.WithStdoutWriter(stdoutWriter),
	)
	if processErr != nil {
		logger.Debugf(ctx, "python mutator process failed: %s", processErr)
	}

	pythonDiagnostics, pythonDiagnosticsErr := loadDiagnosticsFile(diagnosticsPath)
	if pythonDiagnosticsErr != nil {
		logger.Debugf(ctx, "failed to load diagnostics: %s", pythonDiagnosticsErr)
	}

	// if diagnostics file exists, it gives the most descriptive errors
	// if there is any error, we treat it as fatal error, and stop processing
	if pythonDiagnostics.HasError() {
		return dyn.InvalidValue, pythonDiagnostics
	}

	// process can fail without reporting errors in diagnostics file or creating it, for instance,
	// venv doesn't have PyDABs library installed
	if processErr != nil {
		return dyn.InvalidValue, diag.Errorf("python mutator process failed: %sw, use --debug to enable logging", processErr)
	}

	// or we can fail to read diagnostics file, that should always be created
	if pythonDiagnosticsErr != nil {
		return dyn.InvalidValue, diag.Errorf("failed to load diagnostics: %s", pythonDiagnosticsErr)
	}

	output, err := loadOutputFile(rootPath, outputPath)
	if err != nil {
		return dyn.InvalidValue, diag.Errorf("failed to load Python mutator output: %s", err)
	}

	// we pass through pythonDiagnostic because it contains warnings
	return output, pythonDiagnostics
}

func writeInputFile(inputPath string, input dyn.Value) error {
	// we need to marshal dyn.Value instead of bundle.Config to JSON to support
	// non-string fields assigned with bundle variables
	rootConfigJson, err := json.Marshal(input.AsAny())
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	return os.WriteFile(inputPath, rootConfigJson, 0600)
}

func loadOutputFile(rootPath string, outputPath string) (dyn.Value, error) {
	outputFile, err := os.Open(outputPath)
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to open output file: %w", err)
	}

	defer outputFile.Close()

	// we need absolute path because later parts of pipeline assume all paths are absolute
	// and this file will be used as location to resolve relative paths.
	//
	// virtualPath has to stay in rootPath, because locations outside root path are not allowed:
	//
	//   Error: path /var/folders/.../pydabs/dist/*.whl is not contained in bundle root path
	//
	// for that, we pass virtualPath instead of outputPath as file location
	virtualPath, err := filepath.Abs(filepath.Join(rootPath, "__generated_by_pydabs__.yml"))
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to get absolute path: %w", err)
	}

	generated, err := yamlloader.LoadYAML(virtualPath, outputFile)
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to parse output file: %w", err)
	}

	normalized, diagnostic := convert.Normalize(config.Root{}, generated)
	if diagnostic.Error() != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to normalize output: %w", diagnostic.Error())
	}

	// warnings shouldn't happen because output should be already normalized
	// when it happens, it's a bug in the mutator, and should be treated as an error

	for _, d := range diagnostic.Filter(diag.Warning) {
		return dyn.InvalidValue, fmt.Errorf("failed to normalize output: %s", d.Summary)
	}

	return normalized, nil
}

// loadDiagnosticsFile loads diagnostics from a file.
//
// It contains a list of warnings and errors that we should print to users.
//
// If the file doesn't exist, we return an error. We expect the file to always be
// created by the Python mutator, and it's absence means there are integration problems,
// and the diagnostics file was lost. If we treat non-existence as an empty diag.Diagnostics
// we risk loosing errors and warnings.
func loadDiagnosticsFile(path string) (diag.Diagnostics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open diagnostics file: %w", err)
	}

	defer file.Close()

	return parsePythonDiagnostics(file)
}

func createOverrideVisitor(ctx context.Context, phase phase) (merge.OverrideVisitor, error) {
	switch phase {
	case PythonMutatorPhaseLoad:
		return createLoadOverrideVisitor(ctx), nil
	case PythonMutatorPhaseInit:
		return createInitOverrideVisitor(ctx), nil
	default:
		return merge.OverrideVisitor{}, fmt.Errorf("unknown phase: %s", phase)
	}
}

// createLoadOverrideVisitor creates an override visitor for the load phase.
//
// During load, it's only possible to create new resources, and not modify or
// delete existing ones.
func createLoadOverrideVisitor(ctx context.Context) merge.OverrideVisitor {
	resourcesPath := dyn.NewPath(dyn.Key("resources"))
	jobsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"))

	return merge.OverrideVisitor{
		VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
			if isOmitemptyDelete(left) {
				return merge.ErrOverrideUndoDelete
			}

			return fmt.Errorf("unexpected change at %q (delete)", valuePath.String())
		},
		VisitInsert: func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			// insert 'resources' or 'resources.jobs' if it didn't exist before
			if valuePath.Equal(resourcesPath) || valuePath.Equal(jobsPath) {
				return right, nil
			}

			if !valuePath.HasPrefix(jobsPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (insert)", valuePath.String())
			}

			insertResource := len(valuePath) == len(jobsPath)+1

			// adding a property into an existing resource is not allowed, because it changes it
			if !insertResource {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (insert)", valuePath.String())
			}

			log.Debugf(ctx, "Insert value at %q", valuePath.String())

			return right, nil
		},
		VisitUpdate: func(valuePath dyn.Path, left dyn.Value, right dyn.Value) (dyn.Value, error) {
			return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (update)", valuePath.String())
		},
	}
}

// createInitOverrideVisitor creates an override visitor for the init phase.
//
// During the init phase it's possible to create new resources, modify existing
// resources, but not delete existing resources.
func createInitOverrideVisitor(ctx context.Context) merge.OverrideVisitor {
	resourcesPath := dyn.NewPath(dyn.Key("resources"))
	jobsPath := dyn.NewPath(dyn.Key("resources"), dyn.Key("jobs"))

	return merge.OverrideVisitor{
		VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
			if isOmitemptyDelete(left) {
				return merge.ErrOverrideUndoDelete
			}

			if !valuePath.HasPrefix(jobsPath) {
				return fmt.Errorf("unexpected change at %q (delete)", valuePath.String())
			}

			deleteResource := len(valuePath) == len(jobsPath)+1

			if deleteResource {
				return fmt.Errorf("unexpected change at %q (delete)", valuePath.String())
			}

			// deleting properties is allowed because it only changes an existing resource
			log.Debugf(ctx, "Delete value at %q", valuePath.String())

			return nil
		},
		VisitInsert: func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			// insert 'resources' or 'resources.jobs' if it didn't exist before
			if valuePath.Equal(resourcesPath) || valuePath.Equal(jobsPath) {
				return right, nil
			}

			if !valuePath.HasPrefix(jobsPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (insert)", valuePath.String())
			}

			log.Debugf(ctx, "Insert value at %q", valuePath.String())

			return right, nil
		},
		VisitUpdate: func(valuePath dyn.Path, left dyn.Value, right dyn.Value) (dyn.Value, error) {
			if !valuePath.HasPrefix(jobsPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (update)", valuePath.String())
			}

			log.Debugf(ctx, "Update value at %q", valuePath.String())

			return right, nil
		},
	}
}

func isOmitemptyDelete(left dyn.Value) bool {
	// PyDABs can omit empty sequences/mappings in output, because we don't track them as optional,
	// there is no semantic difference between empty and missing, so we keep them as they were before
	// PyDABs deleted them.

	switch left.Kind() {
	case dyn.KindMap:
		return left.MustMap().Len() == 0

	case dyn.KindSequence:
		return len(left.MustSequence()) == 0

	case dyn.KindNil:
		// map/sequence can be nil, for instance, bad YAML like: `foo:<eof>`
		return true

	default:
		return false
	}
}

// detectExecutable lookups Python interpreter in virtual environment, or if not set, in PATH.
func detectExecutable(ctx context.Context, venvPath string) (string, error) {
	if venvPath == "" {
		interpreter, err := python.DetectExecutable(ctx)
		if err != nil {
			return "", err
		}

		return interpreter, nil
	}

	return python.DetectVEnvExecutable(venvPath)
}
