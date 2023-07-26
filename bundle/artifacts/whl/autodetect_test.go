package whl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractModuleName(t *testing.T) {
	moduleName := extractModuleName("./testdata/setup.py")
	assert.Equal(t, "my_test_code", moduleName)
}

func TestExtractModuleNameMinimal(t *testing.T) {
	moduleName := extractModuleName("./testdata/setup_minimal.py")
	assert.Equal(t, "my_test_code", moduleName)
}

func TestExtractModuleNameIncorrect(t *testing.T) {
	moduleName := extractModuleName("./testdata/setup_incorrect.py")
	assert.Contains(t, moduleName, "artifact")
}
