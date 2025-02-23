package python

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config/mutator/paths"

	"github.com/databricks/databricks-sdk-go/logger"
	"github.com/fatih/color"

	"github.com/databricks/cli/libs/python"

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
	//
	// Deprecated, left for backward-compatibility with PyDABs.
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
	//
	// Deprecated, left for backward-compatibility with PyDABs.
	PythonMutatorPhaseInit phase = "init"

	// PythonMutatorPhaseLoadResources is the phase in which YAML configuration was loaded.
	//
	// At this stage, we execute Python code to load resources defined in Python.
	//
	// During this process, Python code can access:
	// - selected deployment target
	// - bundle variable values
	// - variables provided through CLI argument or environment variables
	//
	// The following is not available:
	// - variables referencing other variables are in unresolved format
	//
	// Python code can output YAML referencing variables, and CLI should resolve them.
	//
	// Existing resources can't be removed or modified, and CLI rejects such changes.
	// While it's called 'load_resources', this phase is executed in 'init' phase of mutator pipeline.
	PythonMutatorPhaseLoadResources phase = "load_resources"

	// PythonMutatorPhaseApplyMutators is the phase in which resources defined in YAML or Python
	// are already loaded.
	//
	// At this stage, we execute Python code to mutate resources defined in YAML or Python.
	//
	// During this process, Python code can access:
	// - selected deployment target
	// - bundle variable values
	// - variables provided through CLI argument or environment variables
	//
	// The following is not available:
	// - variables referencing other variables are in unresolved format
	//
	// Python code can output YAML referencing variables, and CLI should resolve them.
	//
	// Resources can't be added or removed, and CLI rejects such changes. Python code is
	// allowed to modify existing resources, but not other parts of bundle configuration.
	PythonMutatorPhaseApplyMutators phase = "apply_mutators"
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

// opts is a common structure for deprecated PyDABs and upcoming Python
// configuration sections
type opts struct {
	enabled bool

	venvPath string

	loadLocations bool
}

type runPythonMutatorOpts struct {
	cacheDir       string
	bundleRootPath string
	pythonPath     string
	loadLocations  bool
}

// getOpts adapts deprecated PyDABs and upcoming Python configuration
// into a common structure.
func getOpts(b *bundle.Bundle, phase phase) (opts, error) {
	experimental := b.Config.Experimental
	if experimental == nil {
		return opts{}, nil
	}

	// using reflect.DeepEquals in case we add more fields
	pydabsEnabled := !reflect.DeepEqual(experimental.PyDABs, config.PyDABs{})
	pythonEnabled := !reflect.DeepEqual(experimental.Python, config.Python{})

	if pydabsEnabled && pythonEnabled {
		return opts{}, errors.New("both experimental/pydabs and experimental/python are enabled, only one can be enabled")
	} else if pydabsEnabled {
		if !experimental.PyDABs.Enabled {
			return opts{}, nil
		}

		// don't execute for phases for 'python' section
		if phase == PythonMutatorPhaseInit || phase == PythonMutatorPhaseLoad {
			return opts{
				enabled:       true,
				venvPath:      experimental.PyDABs.VEnvPath,
				loadLocations: false, // not supported in PyDABs
			}, nil
		} else {
			return opts{}, nil
		}
	} else if pythonEnabled {
		// don't execute for phases for 'pydabs' section
		if phase == PythonMutatorPhaseLoadResources || phase == PythonMutatorPhaseApplyMutators {
			return opts{
				enabled:       true,
				venvPath:      experimental.Python.VEnvPath,
				loadLocations: true,
			}, nil
		} else {
			return opts{}, nil
		}
	} else {
		return opts{}, nil
	}
}

func (m *pythonMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	opts, err := getOpts(b, m.phase)
	if err != nil {
		return diag.Errorf("failed to apply python mutator: %s", err)
	}

	if !opts.enabled {
		return nil
	}

	// mutateDiags is used because Mutate returns 'error' instead of 'diag.Diagnostics'
	var mutateDiags diag.Diagnostics
	mutateDiagsHasError := errors.New("unexpected error")

	err = b.Config.Mutate(func(leftRoot dyn.Value) (dyn.Value, error) {
		pythonPath, err := detectExecutable(ctx, opts.venvPath)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to get Python interpreter path: %w", err)
		}

		cacheDir, err := createCacheDir(ctx)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to create cache dir: %w", err)
		}

		rightRoot, diags := m.runPythonMutator(ctx, leftRoot, runPythonMutatorOpts{
			cacheDir:       cacheDir,
			bundleRootPath: b.BundleRootPath,
			pythonPath:     pythonPath,
			loadLocations:  opts.loadLocations,
		})
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
		cacheDir := filepath.Join(tempDir, "default", "python")

		err := os.MkdirAll(cacheDir, 0o700)
		if err != nil {
			return "", err
		}

		return cacheDir, nil
	}

	return os.MkdirTemp("", "-python")
}

func (m *pythonMutator) runPythonMutator(ctx context.Context, root dyn.Value, opts runPythonMutatorOpts) (dyn.Value, diag.Diagnostics) {
	inputPath := filepath.Join(opts.cacheDir, "input.json")
	outputPath := filepath.Join(opts.cacheDir, "output.json")
	diagnosticsPath := filepath.Join(opts.cacheDir, "diagnostics.json")
	locationsPath := filepath.Join(opts.cacheDir, "locations.json")

	args := []string{
		opts.pythonPath,
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

	if opts.loadLocations {
		args = append(args, "--locations", locationsPath)
	}

	if err := writeInputFile(inputPath, root); err != nil {
		return dyn.InvalidValue, diag.Errorf("failed to write input file: %s", err)
	}

	stderrBuf := bytes.Buffer{}
	stderrWriter := io.MultiWriter(
		newLogWriter(ctx, "stderr: "),
		&stderrBuf,
	)
	stdoutWriter := newLogWriter(ctx, "stdout: ")

	_, processErr := process.Background(
		ctx,
		args,
		process.WithDir(opts.bundleRootPath),
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
	// venv doesn't have 'databricks-bundles' library installed
	if processErr != nil {
		diagnostic := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("python mutator process failed: %q, use --debug to enable logging", processErr),
			Detail:   explainProcessErr(stderrBuf.String()),
		}

		return dyn.InvalidValue, diag.Diagnostics{diagnostic}
	}

	// or we can fail to read diagnostics file, that should always be created
	if pythonDiagnosticsErr != nil {
		return dyn.InvalidValue, diag.Errorf("failed to load diagnostics: %s", pythonDiagnosticsErr)
	}

	locations, err := loadLocationsFile(opts.bundleRootPath, locationsPath)
	if err != nil {
		return dyn.InvalidValue, diag.Errorf("failed to load locations: %s", err)
	}

	output, outputDiags := loadOutputFile(opts.bundleRootPath, outputPath, locations)
	pythonDiagnostics = pythonDiagnostics.Extend(outputDiags)

	// we pass through pythonDiagnostic because it contains warnings
	return output, pythonDiagnostics
}

const pythonInstallExplanation = `Ensure that 'databricks-bundles' is installed in Python environment:

  $ .venv/bin/pip install databricks-bundles

If using a virtual environment, ensure it is specified as the venv_path property in databricks.yml,
or activate the environment before running CLI commands:

  experimental:
    python:
      venv_path: .venv
`

// explainProcessErr provides additional explanation for common errors.
// It's meant to be the best effort, and not all errors are covered.
// Output should be used only used for error reporting.
func explainProcessErr(stderr string) string {
	// implemented in cpython/Lib/runpy.py and portable across Python 3.x, including pypy
	if strings.Contains(stderr, "Error while finding module specification for 'databricks.bundles.build'") {
		summary := color.CyanString("Explanation: ") + "'databricks-bundles' library is not installed in the Python environment.\n"

		return stderr + "\n" + summary + "\n" + pythonInstallExplanation
	}

	return stderr
}

func writeInputFile(inputPath string, input dyn.Value) error {
	// we need to marshal dyn.Value instead of bundle.Config to JSON to support
	// non-string fields assigned with bundle variables
	rootConfigJson, err := json.Marshal(input.AsAny())
	if err != nil {
		return fmt.Errorf("failed to marshal input: %w", err)
	}

	return os.WriteFile(inputPath, rootConfigJson, 0o600)
}

// loadLocationsFile loads locations.json containing source locations for generated YAML.
func loadLocationsFile(bundleRoot, locationsPath string) (*pythonLocations, error) {
	locationsFile, err := os.Open(locationsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return newPythonLocations(), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to open locations file: %w", err)
	}

	defer locationsFile.Close()

	return parsePythonLocations(bundleRoot, locationsFile)
}

func loadOutputFile(rootPath, outputPath string, locations *pythonLocations) (dyn.Value, diag.Diagnostics) {
	outputFile, err := os.Open(outputPath)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to open output file: %w", err))
	}

	defer outputFile.Close()

	return loadOutput(rootPath, outputFile, locations)
}

func loadOutput(rootPath string, outputFile io.Reader, locations *pythonLocations) (dyn.Value, diag.Diagnostics) {
	// we need absolute path because later parts of pipeline assume all paths are absolute
	// and this file will be used as location to resolve relative paths.
	//
	// virtualPath has to stay in bundleRootPath, because locations outside root path are not allowed:
	//
	//   Error: path /var/folders/.../python/dist/*.whl is not contained in bundle root path
	//
	// for that, we pass virtualPath instead of outputPath as file location
	virtualPath, err := filepath.Abs(filepath.Join(rootPath, generatedFileName))
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to get absolute path: %w", err))
	}

	generated, err := yamlloader.LoadYAML(virtualPath, outputFile)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to parse output file: %w", err))
	}

	// paths are resolved relative to locations of their values, if we change location
	// we have to update each path, until we simplify that, we don't update locations
	// for such values, so we don't change how paths are resolved
	//
	// we can remove this once we:
	// - add variable interpolation before and after PythonMutator
	// - implement path normalization (aka path normal form)
	_, err = paths.VisitJobPaths(generated, func(p dyn.Path, kind paths.PathKind, v dyn.Value) (dyn.Value, error) {
		putPythonLocation(locations, p, v.Location())
		return v, nil
	})
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to update locations: %w", err))
	}

	// generated has dyn.Location as if it comes from generated YAML file
	// earlier we loaded locations.json with source locations in Python code
	generatedWithLocations, err := mergePythonLocations(generated, locations)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to update locations: %w", err))
	}

	return strictNormalize(config.Root{}, generatedWithLocations)
}

func strictNormalize(dst any, generated dyn.Value) (dyn.Value, diag.Diagnostics) {
	normalized, diags := convert.Normalize(dst, generated)

	// warnings shouldn't happen because output should be already normalized
	// when it happens, it's a bug in the mutator, and should be treated as an error

	strictDiags := diag.Diagnostics{}

	for _, d := range diags {
		if d.Severity == diag.Warning {
			d.Severity = diag.Error
		}

		strictDiags = strictDiags.Append(d)
	}

	return normalized, strictDiags
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
		return createLoadResourcesOverrideVisitor(ctx), nil
	case PythonMutatorPhaseInit:
		return createInitOverrideVisitor(ctx, insertResourceModeAllow), nil
	case PythonMutatorPhaseLoadResources:
		return createLoadResourcesOverrideVisitor(ctx), nil
	case PythonMutatorPhaseApplyMutators:
		return createInitOverrideVisitor(ctx, insertResourceModeDisallow), nil
	default:
		return merge.OverrideVisitor{}, fmt.Errorf("unknown phase: %s", phase)
	}
}

// createLoadResourcesOverrideVisitor creates an override visitor for the load_resources phase.
//
// During load_resources, it's only possible to create new resources, and not modify or
// delete existing ones.
func createLoadResourcesOverrideVisitor(ctx context.Context) merge.OverrideVisitor {
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
		VisitUpdate: func(valuePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
			return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (update)", valuePath.String())
		},
	}
}

// insertResourceMode controls whether createInitOverrideVisitor allows or disallows inserting new resources.
type insertResourceMode int

const (
	insertResourceModeDisallow insertResourceMode = iota
	insertResourceModeAllow    insertResourceMode = iota
)

// createInitOverrideVisitor creates an override visitor for the init phase.
//
// During the init phase it's possible to create new resources, modify existing
// resources, but not delete existing resources.
//
// If mode is insertResourceModeDisallow, it matching expected behaviour of apply_mutators
func createInitOverrideVisitor(ctx context.Context, mode insertResourceMode) merge.OverrideVisitor {
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

			insertResource := len(valuePath) == len(jobsPath)+1
			if mode == insertResourceModeDisallow && insertResource {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (insert)", valuePath.String())
			}

			log.Debugf(ctx, "Insert value at %q", valuePath.String())

			return right, nil
		},
		VisitUpdate: func(valuePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
			if !valuePath.HasPrefix(jobsPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (update)", valuePath.String())
			}

			log.Debugf(ctx, "Update value at %q", valuePath.String())

			return right, nil
		},
	}
}

func isOmitemptyDelete(left dyn.Value) bool {
	// Python output can omit empty sequences/mappings, because we don't track them as optional,
	// there is no semantic difference between empty and missing, so we keep them as they were before
	// Python mutator deleted them.

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
