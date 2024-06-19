package yamlloader_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestYAMLMix01(t *testing.T) {
	file := "testdata/mix_01.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	assert.True(t, self.Get("base_address").IsAnchor())
	assert.False(t, self.Get("office_address").IsAnchor())
}

func TestYAMLMix02(t *testing.T) {
	file := "testdata/mix_02.yml"
	self := loadYAML(t, file)
	assert.NotEqual(t, dyn.NilValue, self)

	assert.True(t, self.Get("base_colors").IsAnchor())
	assert.False(t, self.Get("theme").IsAnchor())
}
