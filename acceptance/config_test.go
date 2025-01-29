package acceptance_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/require"
)

const configFilename = "test.toml"

var (
	configCache map[string]TestConfig
	configMutex sync.Mutex
)

type TestConfig struct {
	// Place to describe what's wrong with this test. Does not affect how the test is run.
	Badness string

	// Which OSes the test is enabled on. Each string is compared against runtime.GOOS.
	// If absent, default to true.
	GOOS map[string]bool

	// List of additional replacements to apply on this test.
	// Old is a regexp, New is a replacement expression.
	Repls []testdiff.Replacement

	// List of server stubs to load.
	Server []testserver.Stub
}

// FindConfig finds the closest config file.
func FindConfig(t *testing.T, dir string) (string, bool) {
	shared := false
	for {
		path := filepath.Join(dir, configFilename)
		_, err := os.Stat(path)

		if err == nil {
			return path, shared
		}

		shared = true

		if dir == "" || dir == "." {
			break
		}

		if os.IsNotExist(err) {
			dir = filepath.Dir(dir)
			continue
		}

		t.Fatalf("Error while reading %s: %s", path, err)
	}

	t.Fatal("Config not found: " + configFilename)
	return "", shared
}

// LoadConfig loads the config file. Non-leaf configs are cached.
func LoadConfig(t *testing.T, dir string) (TestConfig, string) {
	path, leafConfig := FindConfig(t, dir)

	if leafConfig {
		return DoLoadConfig(t, path), path
	}

	configMutex.Lock()
	defer configMutex.Unlock()

	if configCache == nil {
		configCache = make(map[string]TestConfig)
	}

	result, ok := configCache[path]
	if ok {
		return result, path
	}

	result = DoLoadConfig(t, path)
	configCache[path] = result
	return result, path
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
