package javascript

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

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"

	"github.com/databricks/databricks-sdk-go/logger"

	"github.com/databricks/cli/bundle/env"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/process"
)

type phase string

const (
	// JavaScriptMutatorPhaseLoadResources is the phase in which YAML configuration was loaded.
	//
	// At this stage, we execute JavaScript code to load resources defined in JavaScript.
	//
	// During this process, JavaScript code can access:
	// - selected deployment target
	// - bundle variable values
	// - variables provided through CLI argument or environment variables
	//
	// The following is not available:
	// - variables referencing other variables are in unresolved format
	//
	// JavaScript code can output YAML referencing variables, and CLI should resolve them.
	//
	// Existing resources can't be removed or modified, and CLI rejects such changes.
	JavaScriptMutatorPhaseLoadResources phase = "load_resources"
)

type javaScriptMutator struct {
	phase phase
}

func JavaScriptMutator(phase phase) bundle.Mutator {
	return &javaScriptMutator{
		phase: phase,
	}
}

func (m *javaScriptMutator) Name() string {
	return fmt.Sprintf("JavaScriptMutator(%s)", m.phase)
}

type opts struct {
	enabled bool

	loadLocations bool
}

type runJavaScriptMutatorOpts struct {
	cacheDir       string
	bundleRootPath string
	nodePath       string
	loadLocations  bool
}

func getOpts(b *bundle.Bundle, phase phase) (opts, error) {
	experimental := b.Config.Experimental
	if experimental == nil {
		return opts{}, nil
	}

	javascriptEnabled := !reflect.DeepEqual(experimental.JavaScript, config.JavaScript{})

	if javascriptEnabled {
		if phase == JavaScriptMutatorPhaseLoadResources {
			return opts{
				enabled:       true,
				loadLocations: true,
			}, nil
		} else {
			return opts{}, nil
		}
	} else {
		return opts{}, nil
	}
}

func (m *javaScriptMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	opts, err := getOpts(b, m.phase)
	if err != nil {
		return diag.Errorf("failed to apply javascript mutator: %s", err)
	}

	if !opts.enabled {
		return nil
	}

	// Don't run any arbitrary code when restricted execution is enabled.
	if _, ok := env.RestrictedExecution(ctx); ok {
		return diag.Errorf("Running JavaScript code is not allowed when DATABRICKS_BUNDLE_RESTRICTED_CODE_EXECUTION is set")
	}

	// mutateDiags is used because Mutate returns 'error' instead of 'diag.Diagnostics'
	var mutateDiags diag.Diagnostics
	var result applyJavaScriptOutputResult
	mutateDiagsHasError := errors.New("unexpected error")

	err = b.Config.Mutate(func(leftRoot dyn.Value) (dyn.Value, error) {
		nodePath, err := detectExecutable(ctx)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to get Node.js path: %w", err)
		}

		cacheDir, err := createCacheDir(ctx)
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to create cache dir: %w", err)
		}

		rightRoot, diags := m.runJavaScriptMutator(ctx, leftRoot, runJavaScriptMutatorOpts{
			cacheDir:       cacheDir,
			bundleRootPath: b.BundleRootPath,
			nodePath:       nodePath,
			loadLocations:  opts.loadLocations,
		})
		mutateDiags = diags
		if diags.HasError() {
			return dyn.InvalidValue, mutateDiagsHasError
		}

		newRoot, result0, err := applyJavaScriptOutput(leftRoot, rightRoot)
		result = result0
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("internal error when merging output of JavaScript mutator: %w", err)
		}

		for _, resourceKey := range result.AddedResources.ToArray() {
			log.Debugf(ctx, "added resource at 'resources.%s.%s'", resourceKey.Type, resourceKey.Name)
		}

		for _, resourceKey := range result.UpdatedResources.ToArray() {
			log.Debugf(ctx, "updated resource at 'resources.%s.%s'", resourceKey.Type, resourceKey.Name)
		}

		for _, resourceKey := range result.DeletedResources.ToArray() {
			log.Debugf(ctx, "deleted resource at 'resources.%s.%s'", resourceKey.Type, resourceKey.Name)
		}

		if !result.DeletedResources.IsEmpty() {
			return dyn.InvalidValue, fmt.Errorf("unexpected deleted resources: %s", result.DeletedResources.ToArray())
		}

		if !result.AddedResources.IsEmpty() && m.phase != JavaScriptMutatorPhaseLoadResources {
			return dyn.InvalidValue, fmt.Errorf("unexpected added resources: %s", result.AddedResources.ToArray())
		}

		if !result.UpdatedResources.IsEmpty() && m.phase == JavaScriptMutatorPhaseLoadResources {
			return dyn.InvalidValue, fmt.Errorf("unexpected updated resources: %s", result.UpdatedResources.ToArray())
		}

		return newRoot, nil
	})

	// we can precisely track resources that are added/updated, so sum doesn't double-count
	b.Metrics.JavaScriptUpdatedResourcesCount += int64(result.UpdatedResources.Size())
	b.Metrics.JavaScriptAddedResourcesCount += int64(result.AddedResources.Size())

	if err == mutateDiagsHasError {
		if !mutateDiags.HasError() {
			panic("mutateDiags has no error, but error is expected")
		}

		return mutateDiags
	} else {
		mutateDiags = mutateDiags.Extend(diag.FromErr(err))
	}

	if mutateDiags.HasError() {
		return mutateDiags
	}

	resourcemutator.NormalizeAndInitializeResources(ctx, b, result.AddedResources)
	if logdiag.HasError(ctx) {
		return mutateDiags
	}

	resourcemutator.NormalizeResources(ctx, b, result.UpdatedResources)
	return mutateDiags
}

func createCacheDir(ctx context.Context) (string, error) {
	// support the same env variable as in b.LocalStateDir
	if tempDir, exists := env.TempDir(ctx); exists {
		// use 'default' as target name
		cacheDir := filepath.Join(tempDir, "default", "javascript")

		err := os.MkdirAll(cacheDir, 0o700)
		if err != nil {
			return "", err
		}

		return cacheDir, nil
	}

	return os.MkdirTemp("", "-javascript")
}

func (m *javaScriptMutator) runJavaScriptMutator(ctx context.Context, root dyn.Value, opts runJavaScriptMutatorOpts) (dyn.Value, diag.Diagnostics) {
	inputPath := filepath.Join(opts.cacheDir, "input.json")
	outputPath := filepath.Join(opts.cacheDir, "output.json")
	diagnosticsPath := filepath.Join(opts.cacheDir, "diagnostics.json")
	locationsPath := filepath.Join(opts.cacheDir, "locations.json")

	// Get the JavaScript file path from the config
	experimental := root.Get("experimental")
	if experimental.Kind() == dyn.KindInvalid {
		return dyn.InvalidValue, diag.Errorf("experimental configuration not found")
	}

	jsConfig := experimental.Get("javascript")
	if jsConfig.Kind() == dyn.KindInvalid {
		return dyn.InvalidValue, diag.Errorf("experimental.javascript configuration not found")
	}

	resources := jsConfig.Get("resources")
	if resources.Kind() == dyn.KindInvalid || resources.Kind() != dyn.KindSequence {
		return dyn.InvalidValue, diag.Errorf("experimental.javascript.resources must be an array")
	}

	resourcesSeq := resources.MustSequence()
	if len(resourcesSeq) == 0 {
		return dyn.InvalidValue, diag.Errorf("experimental.javascript.resources must contain at least one file path")
	}

	jsFilePath := "/Users/fabian.jakobs/Workspaces/cli/experimental/typescript/dist/src/cli.js"

	args := []string{
		opts.nodePath,
		jsFilePath,
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
		logger.Debugf(ctx, "javascript mutator process failed: %s", processErr)
	}

	javaScriptDiagnostics, javaScriptDiagnosticsErr := loadDiagnosticsFile(diagnosticsPath)
	if javaScriptDiagnosticsErr != nil {
		logger.Debugf(ctx, "failed to load diagnostics: %s", javaScriptDiagnosticsErr)
	}

	// if diagnostics file exists, it gives the most descriptive errors
	// if there is any error, we treat it as fatal error, and stop processing
	if javaScriptDiagnostics.HasError() {
		return dyn.InvalidValue, javaScriptDiagnostics
	}

	// process can fail without reporting errors in diagnostics file or creating it
	if processErr != nil {
		diagnostic := diag.Diagnostic{
			Severity: diag.Error,
			Summary:  fmt.Sprintf("javascript mutator process failed: %q, use --debug to enable logging", processErr),
			Detail:   stderrBuf.String(),
		}

		return dyn.InvalidValue, diag.Diagnostics{diagnostic}
	}

	// or we can fail to read diagnostics file, that should always be created
	if javaScriptDiagnosticsErr != nil {
		return dyn.InvalidValue, diag.Errorf("failed to load diagnostics: %s", javaScriptDiagnosticsErr)
	}

	locations, err := loadLocationsFile(opts.bundleRootPath, locationsPath)
	if err != nil {
		return dyn.InvalidValue, diag.Errorf("failed to load locations: %s", err)
	}

	output, outputDiags := loadOutputFile(opts.bundleRootPath, outputPath, locations)
	javaScriptDiagnostics = javaScriptDiagnostics.Extend(outputDiags)

	// we pass through javaScriptDiagnostic because it contains warnings
	return output, javaScriptDiagnostics
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
func loadLocationsFile(bundleRoot, locationsPath string) (*javaScriptLocations, error) {
	locationsFile, err := os.Open(locationsPath)
	if errors.Is(err, fs.ErrNotExist) {
		return newJavaScriptLocations(), nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to open locations file: %w", err)
	}

	defer locationsFile.Close()

	return parseJavaScriptLocations(bundleRoot, locationsFile)
}

func loadOutputFile(rootPath, outputPath string, locations *javaScriptLocations) (dyn.Value, diag.Diagnostics) {
	outputFile, err := os.Open(outputPath)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to open output file: %w", err))
	}

	defer outputFile.Close()

	return loadOutput(rootPath, outputFile, locations)
}

func loadOutput(rootPath string, outputFile io.Reader, locations *javaScriptLocations) (dyn.Value, diag.Diagnostics) {
	// we need absolute path because later parts of pipeline assume all paths are absolute
	// and this file will be used as location to resolve relative paths.
	virtualPath, err := filepath.Abs(filepath.Join(rootPath, generatedFileName))
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to get absolute path: %w", err))
	}

	generated, err := yamlloader.LoadYAML(virtualPath, outputFile)
	if err != nil {
		return dyn.InvalidValue, diag.FromErr(fmt.Errorf("failed to parse output file: %w", err))
	}

	// generated has dyn.Location as if it comes from generated YAML file
	// earlier we loaded locations.json with source locations in JavaScript code
	generatedWithLocations, err := mergeJavaScriptLocations(generated, locations)
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
func loadDiagnosticsFile(path string) (diag.Diagnostics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open diagnostics file: %w", err)
	}

	defer file.Close()

	return parseJavaScriptDiagnostics(file)
}

// detectExecutable lookups Node.js interpreter in PATH.
func detectExecutable(ctx context.Context) (string, error) {
	// Try to find node in PATH
	_, processErr := process.Background(
		ctx,
		[]string{"node", "--version"},
	)
	if processErr != nil {
		return "", fmt.Errorf("node not found in PATH: %w", processErr)
	}

	return "node", nil
}
