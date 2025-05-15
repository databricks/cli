package internal

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"dario.cat/mergo"
	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/cli/libs/testserver"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/require"
)

const configFilename = "test.toml"

type TestConfig struct {
	// Place to describe what's wrong with this test. Does not affect how the test is run.
	Badness *string

	// Which OSes the test is enabled on. Each string is compared against runtime.GOOS.
	// If absent, default to true.
	GOOS map[string]bool

	// Which Clouds the test is enabled on. Allowed values: "aws", "azure", "gcp".
	// If absent, default to true.
	// Only checked if CLOUD_ENV is not empty.
	CloudEnvs map[string]bool

	// If true, run this test when running locally with a testserver
	Local *bool

	// If true, run this test when running with cloud env configured
	Cloud *bool

	// If true, run this test when running with cloud env configured and -short is not passed
	// This also sets -tail when -v is passed.
	CloudSlow *bool

	// If true and Cloud=true, run this test only if unity catalog is available in the cloud environment
	RequiresUnityCatalog *bool

	// If true and Cloud=true, run this test only if a default test cluster is available in the cloud environment
	RequiresCluster *bool

	// If true and Cloud=true, run this test only if a default warehouse is available in the cloud environment
	RequiresWarehouse *bool

	// List of additional replacements to apply on this test.
	// Old is a regexp, New is a replacement expression.
	Repls []testdiff.Replacement

	// List of server stubs to load. Example configuration:
	//
	// [[Server]]
	// Pattern = "POST /api/2.2/jobs/create"
	// Response.Body = '''
	// {
	// 	"job_id": 1111
	// }
	// '''
	Server []ServerStub

	// Record the requests made to the server and write them as output to
	// out.requests.txt
	RecordRequests *bool

	// List of request headers to include when recording requests.
	IncludeRequestHeaders []string

	// List of gitignore patterns to ignore when checking output files
	Ignore []string

	CompiledIgnoreObject *ignore.GitIgnore

	// Environment variables
	// If the same variable is defined both in Env and EnvMatrix, the one in EnvMatrix takes precedence
	// regardless of which config file it is defined in.
	// Note, keys are sorted alphabetically rather in order they are defined. This matter when one key reference another (same for EnvMatrix).
	Env map[string]string

	// Environment variables matrix.
	// For each key you can specify zero, one or more values.
	// If you specify zero, the key is omitted, as if it was not defined at all.
	// Otherwise, for each value, you will get a new test with that environment variable
	// set to that value (and replacement configured to match the value).
	// If there are multiple variables defined, all combinations of tests are created,
	// similar to github actions matrix strategy.
	EnvMatrix map[string][]string

	// List of keys for which to do string replacement value -> [KEY]. If not set, defaults to true.
	EnvRepl map[string]bool

	// Maximum amount of time the test is allowed to run, will be killed after that.
	Timeout time.Duration

	// On windows/local run, max(Timeout, TimeoutWindows) is used for timeout
	TimeoutWindows time.Duration

	// For cloud tests, max(Timeout, TimeoutCloud) is used for timeout
	// For cloud+windows tests, max(Timeout, TimeoutWindows, TimeoutCloud) is used for timeout
	TimeoutCloud time.Duration

	// Maps a name (arbitrary string, can be used to override/disable setting in a child config) to a mapping that specifies how
	// to update databricks.yml (or other file designated by BundleConfigTarget).
	// Example:
	// BundleConfig.header.bundle.name = "test-bundle"
	// This overwrite bundle.name in the databricks.yml, so you can omit adding """
	// bundle:
	//   name: somename
	// """ to every test.
	// If child config wants to disable or override this, they can simply do
	// BundleConfig.header = ""
	BundleConfig map[string]any

	// Target config for BundleConfig updates. Empty string disables BundleConfig updates.
	// Null means "databricks.yml"
	BundleConfigTarget *string

	// To be added:
	// BundleConfigMatrix is to BundleConfig what EnvMatrix is to Env
	// It creates different tests for each possible configuration update.
	// BundleConfigMatrix map[string][]any
}

type ServerStub struct {
	// The HTTP method and path to match. Examples:
	// 1. /api/2.0/clusters/list (matches all methods)
	// 2. GET /api/2.0/clusters/list
	Pattern string

	// The response body to return.
	Response testserver.Response

	// Artificial delay to simulate slow responses.
	// Configure as "1ms", "2s", "3m", etc.
	// See [time.ParseDuration] for details.
	Delay time.Duration
}

// FindConfigs finds all the config relevant for this test,
// ordered from the most outermost (at acceptance/) to current test directory (identified by dir).
// Argument dir must be a relative path from the root of acceptance tests (<project_root>/acceptance/).
func FindConfigs(t *testing.T, dir string) []string {
	var configs []string
	for {
		path := filepath.Join(dir, configFilename)
		_, err := os.Stat(path)

		if err == nil {
			configs = append(configs, path)
		}

		if dir == "" || dir == "." {
			break
		}

		dir = filepath.Dir(dir)

		if err == nil || os.IsNotExist(err) {
			continue
		}

		t.Fatalf("Error while reading %s: %s", path, err)
	}

	slices.Reverse(configs)
	return configs
}

// LoadConfig loads the config file. Non-leaf configs are cached.
func LoadConfig(t *testing.T, dir string) (TestConfig, string) {
	configs := FindConfigs(t, dir)

	if len(configs) == 0 {
		return TestConfig{}, "(no config)"
	}

	result := DoLoadConfig(t, configs[0])

	for _, cfgName := range configs[1:] {
		cfg := DoLoadConfig(t, cfgName)
		err := mergo.Merge(
			&result,
			cfg,
			mergo.WithOverride,
			mergo.WithoutDereference,
			mergo.WithAppendSlice,
			mergo.WithTransformers(mapTransformer{}),
		)
		if err != nil {
			t.Fatalf("Error during config merge: %s: %s", cfgName, err)
		}
	}

	result.CompiledIgnoreObject = ignore.CompileIgnoreLines(result.Ignore...)

	return result, strings.Join(configs, ", ")
}

func DoLoadConfig(t *testing.T, path string) TestConfig {
	bytes, err := os.ReadFile(path)
	require.NoError(t, err, "Failed to read test config %s: %s", path, err)

	var config TestConfig
	meta, err := toml.Decode(string(bytes), &config)
	require.NoError(t, err, "Failed to parse config %s", path)

	keys := meta.Undecoded()
	for ind, key := range keys {
		if len(key) > 0 && key[0] == "BundleConfig" {
			continue
		}
		t.Errorf("Undecoded key in %s[%d]: %#v", path, ind, key)
	}

	return config
}

// mapTransformer is a mergo transformer that merges two maps
// by overriding values in the destination map with values from the source map.
//
// In our case, source map is located in test directory, and destination map is located
// in a parent directory.
type mapTransformer struct{}

func (t mapTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	if typ.Kind() == reflect.Map {
		return func(dst, src reflect.Value) error {
			if dst.IsNil() {
				dst.Set(reflect.MakeMap(typ))
			}

			return mergo.Merge(dst.Addr().Interface(), src.Interface(), mergo.WithOverride)
		}
	}

	return nil
}

// This function takes EnvMatrix and expands into a slice of environment configurations.
// Each environment configuration is a slice of env vars in standard Golang format.
// For example,
//
//	input: {"KEY": ["A", "B"], "OTHER": ["VALUE"]}
//
// output: [["KEY=A", "OTHER=VALUE"], ["KEY=B", "OTHER=VALUE"]]
//
// If any entries is an empty list, that variable is dropped from the matrix before processing.
func ExpandEnvMatrix(matrix map[string][]string) [][]string {
	result := [][]string{{}}

	if len(matrix) == 0 {
		return result
	}

	// Filter out keys with empty value slices
	filteredMatrix := make(map[string][]string)
	for key, values := range matrix {
		if len(values) > 0 {
			filteredMatrix[key] = values
		}
	}

	if len(filteredMatrix) == 0 {
		return result
	}

	keys := make([]string, 0, len(filteredMatrix))
	for key := range filteredMatrix {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Build an expansion of all combinations.
	// At each step we look at a given key and append each possible value to each
	// possible result accumulated up to this point.

	for _, key := range keys {
		values := filteredMatrix[key]
		var newResult [][]string

		for _, env := range result {
			for _, value := range values {
				newEnv := make([]string, len(env)+1)
				copy(newEnv, env)
				newEnv[len(env)] = key + "=" + value
				newResult = append(newResult, newEnv)
			}
		}

		result = newResult
	}

	return result
}
