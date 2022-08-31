package configure

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/tests"
	"github.com/stretchr/testify/assert"
	"gopkg.in/ini.v1"
)

func assertKeyValueInSection(t *testing.T, section *ini.Section, keyName, expectedValue string) {
	key, err := section.GetKey(keyName)
	assert.NoError(t, err)
	assert.Equal(t, key.Value(), expectedValue)
}

func setup(t *testing.T) string {
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}
	tests.SetTestEnv(t, homeEnvVar, tempHomeDir)
	tests.SetTestEnv(t, "DATABRICKS_CONFIG_FILE", "")
	return tempHomeDir
}

func TestDefaultConfigure(t *testing.T) {
	ctx := context.Background()
	tempHomeDir := setup(t)

	root.RootCmd.SetArgs([]string{"configure", "-H", "host", "-t", "token"})

	err := root.RootCmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	cfgPath := filepath.Join(tempHomeDir, ".databrickscfg")
	_, err = os.Stat(cfgPath)
	assert.NoError(t, err)

	cfg, err := ini.Load(cfgPath)
	assert.NoError(t, err)

	defaultSection, err := cfg.GetSection("DEFAULT")
	assert.NoError(t, err)

	assertKeyValueInSection(t, defaultSection, "host", "host")
	assertKeyValueInSection(t, defaultSection, "token", "token")
}

func TestConfigFileFromEnv(t *testing.T) {
	ctx := context.Background()
	tempHomeDir := setup(t)
	cfgFileDir := filepath.Join(tempHomeDir, "test")
	tests.SetTestEnv(t, "DATABRICKS_CONFIG_FILE", cfgFileDir)

	root.RootCmd.SetArgs([]string{"configure", "-H", "host", "-t", "token"})

	err := root.RootCmd.ExecuteContext(ctx)
	assert.NoError(t, err)

	cfgPath := filepath.Join(cfgFileDir, ".databrickscfg")
	_, err = os.Stat(cfgPath)
	assert.NoError(t, err)

	cfg, err := ini.Load(cfgPath)
	assert.NoError(t, err)

	defaultSection, err := cfg.GetSection("DEFAULT")
	assert.NoError(t, err)

	assertKeyValueInSection(t, defaultSection, "host", "host")
	assertKeyValueInSection(t, defaultSection, "token", "token")
}
