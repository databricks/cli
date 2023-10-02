package yamlloader_test

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestYAMLMix01(t *testing.T) {
	file := "testdata/mix_01.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, config.NilValue, self)
}

func TestYAMLMix02(t *testing.T) {
	file := "testdata/mix_02.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, config.NilValue, self)
}
