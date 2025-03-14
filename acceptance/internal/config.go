package internal

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

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

	// If true, run this test when running locally with a testserver
	Local *bool

	// If true, run this test when running with cloud env configured
	Cloud *bool

	// If true, run this test when running with cloud env configured and -short is not passed
	// This also sets -tail when -v is passed.
	CloudSlow *bool

	// If true and Cloud=true, run this test only if unity catalog is available in the cloud environment
	RequiresUnityCatalog *bool

	// List of additional replacements to apply on this test.
	// Old is a regexp, New is a replacement expression.
	Repls []testdiff.Replacement

	// List of server stubs to load. Example configuration:
	//
	// [[Server]]
	// Pattern = "POST /api/2.1/jobs/create"
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

	EnvMatrix map[string][]string
}

type ServerStub struct {
	// The HTTP method and path to match. Examples:
	// 1. /api/2.0/clusters/list (matches all methods)
	// 2. GET /api/2.0/clusters/list
	Pattern string

	// The response body to return.
	Response testserver.Response

	// Artificial delay in seconds to simulate slow responses.
	DelaySeconds *float64
}

// FindConfigs finds all the config relevant for this test,
// ordered from the most outermost (at acceptance/) to current test directory (identified by dir).
// Argument dir must be a relative path from the root of acceptance tests (<project_root>/acceptance/).
func FindConfigs(t *testing.T, dir string) []string {
	configs := []string{}
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
		err := mergo.Merge(&result, cfg, mergo.WithOverride, mergo.WithoutDereference, mergo.WithAppendSlice)
		if err != nil {
			t.Fatalf("Error during config merge: %s: %s", cfgName, err)
		}
	}

	result.CompiledIgnoreObject = ignore.CompileIgnoreLines(result.Ignore...)

	return result, strings.Join(configs, ", ")
}

func DoLoadConfig(t *testing.T, path string) TestConfig {
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read config: %s", err)
	}

	var config TestConfig
	meta, err := toml.Decode(string(bytes), &config)
	require.NoError(t, err)

	keys := meta.Undecoded()
	if len(keys) > 0 {
		t.Fatalf("Undecoded keys in %s: %#v", path, keys)
	}

	return config
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
	if len(matrix) == 0 {
		return nil
	}

	// Filter out keys with empty value slices
	filteredMatrix := make(map[string][]string)
	for key, values := range matrix {
		if len(values) > 0 {
			filteredMatrix[key] = values
		}
	}

	if len(filteredMatrix) == 0 {
		// Everything was filtered out
		return nil
	}

	// Get all keys and sort them for deterministic output
	keys := make([]string, 0, len(filteredMatrix))
	for key := range filteredMatrix {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := [][]string{{}}

	// For each key, expand the current result with all possible values for that key
	for _, key := range keys {
		values := filteredMatrix[key]
		var newResult [][]string

		// For each existing environment in the result
		for _, env := range result {
			// For each possible value of the current key
			for _, value := range values {
				// Create a new environment with the current key=value added
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
